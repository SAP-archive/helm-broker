//go:build integration
// +build integration

package integration_test

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v6"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	accessKey    = "MFMJRX021PJLB5V0Q0U1"
	accessSecret = "t3fETPMs7Ca6G5wq2bogJJiqocMcxaNCwOC3aOvj"
	minioIP      = "127.0.0.1"
	minioPort    = "9000"

	addonS3Source = "testdata/addons-s3.tgz"
	bucketName    = "addon-test"
	bucketDir     = "addons"
	bucketRegion  = "us-east-1"
)

type minioServer struct {
	t            *testing.T
	minioProcess *exec.Cmd
	client       *minio.Client

	addonSource string
	minioTmp    string
}

func runMinioServer(t *testing.T, tmpDir string) (*minioServer, error) {
	// create minio client to communicate with minio server
	minioClient, err := minio.New(fmt.Sprintf("%s:%s", minioIP, minioPort), accessKey, accessSecret, false)
	if err != nil {
		return nil, err
	}

	minioSrv := &minioServer{
		t:           t,
		client:      minioClient,
		addonSource: addonS3Source,
		minioTmp:    fmt.Sprintf("%s/minio-%s", strings.TrimRight(tmpDir, "/"), rand.String(4)),
	}

	// set environment for minio server such as key/secret
	err = minioSrv.setEnv()
	if err != nil {
		return minioSrv, err
	}

	// start minio server process
	err = minioSrv.startMinioServer()
	if err != nil {
		return minioSrv, err
	}

	// wait until minio is ready
	err = minioSrv.waitMinioIsReady()
	if err != nil {
		return minioSrv, err
	}

	// create test bucket
	err = minioSrv.createBucket()
	if err != nil {
		return minioSrv, err
	}

	// fill the bucket with files
	err = minioSrv.fillBucket()
	if err != nil {
		return minioSrv, err
	}

	return minioSrv, nil
}

func (m *minioServer) setEnv() error {
	err := os.Setenv("MINIO_ACCESS_KEY", accessKey)
	if err != nil {
		return err
	}

	err = os.Setenv("MINIO_SECRET_KEY", accessSecret)
	if err != nil {
		return err
	}

	return nil
}

func (m *minioServer) startMinioServer() error {
	minioSrvTmp := fmt.Sprintf("%s/server", m.minioTmp)
	m.t.Logf("Run minio server (tmpDir: %s)", minioSrvTmp)

	if err := os.Mkdir(m.minioTmp, 0700); err != nil {
		return errors.Wrap(err, "while creating minio tmp directory")
	}

	cmd := exec.Command("minio", "server", fmt.Sprintf("--address=:%s", minioPort), minioSrvTmp)
	err := cmd.Start()
	if err != nil {
		return errors.Wrap(err, "while starting minio server")
	}

	m.minioProcess = cmd
	return nil
}

func (m *minioServer) waitMinioIsReady() error {
	m.t.Log("Wait for ready minio process")
	err := wait.Poll(1*time.Second, time.Minute, func() (done bool, err error) {
		list, err := m.client.ListBuckets()
		if err != nil {
			m.t.Logf("cannot list minio buckets (%s), minio is not ready, retry...", err)
			return false, nil
		}
		if len(list) != 0 {
			m.t.Logf("bucket list is not empty (%d items), trying clear minio", len(list))
			err = m.clearMinio()
			if err != nil {
				m.t.Logf("error when trying to clear minio: %s", err)
			}
			return false, nil
		}

		m.t.Log("Minio server is ready")
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "while waiting for minio server")
	}

	return nil
}

func (m *minioServer) stopMinioServer() {
	m.t.Log("Stop minio server")

	err := wait.Poll(1*time.Second, 20*time.Second, func() (done bool, err error) {
		err = m.minioProcess.Process.Kill()
		if err != nil && !strings.Contains(err.Error(), "process already finished") {
			m.t.Logf("Cannot kill the process: %d: %s", m.minioProcess.Process.Pid, err)
			return false, err
		}
		state, err := m.minioProcess.Process.Wait()
		if err != nil {
			if strings.Contains(err.Error(), "no child processes") {
				m.t.Log("There is no child processes")
				return true, nil
			}
			m.t.Logf("command process wait error: %s", err)
			return false, err
		}
		if state.Exited() {
			m.t.Log("Process exited")
			return true, nil
		}

		m.t.Logf("process minio (%d) still exist", m.minioProcess.Process.Pid)
		return false, nil
	})

	if err != nil {
		m.t.Logf("Process cannot be stop: %s. Stop minio server process manually.", err)
	}

	m.t.Log("Remove minio server temporary directory")
	if err := m.removeContents(m.minioTmp); err != nil {
		m.t.Logf("Cannot remove minio temp dir content (%s): %s. Remove it manually.", m.minioTmp, err)
	}
	if err := os.Remove(m.minioTmp); err != nil {
		m.t.Logf("Cannot remove minio temp dir (%s): %s. Remove it manually.", m.minioTmp, err)
	}
}

func (m *minioServer) removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *minioServer) clearMinio() error {
	m.t.Log("Clear minio content")

	doneList := make(<-chan struct{})
	list, err := m.client.ListBuckets()

	for _, bucket := range list {
		m.t.Logf("Remove bucket %s", bucket.Name)
		objList := m.client.ListObjects(bucket.Name, "", true, doneList)

		for obj := range objList {
			m.t.Logf(" - remove object %q", obj.Key)
			err = m.client.RemoveObject(bucket.Name, obj.Key)
			if err != nil {
				return err
			}
		}

		m.t.Logf("Bucket %s is clear. Removing.", bucket.Name)
		err = m.client.RemoveBucket(bucket.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *minioServer) createBucket() error {
	m.t.Logf("Create bucket: %q", bucketName)

	err := m.client.MakeBucket(bucketName, bucketRegion)
	if err != nil {
		return errors.Wrap(err, "while creating bucket")
	}

	err = wait.Poll(1*time.Second, 20*time.Second, func() (done bool, err error) {
		_, err = m.client.BucketExists(bucketName)
		if err != nil {
			m.t.Logf("bucket %q not exist (%s). retry...", bucketName, err)
			return false, nil
		}

		m.t.Logf("Bucket %q exist", bucketName)
		return true, nil
	})
	if err != nil {
		return errors.Wrap(err, "while waiting for bucket")
	}

	return nil
}

func (m *minioServer) fillBucket() error {
	m.t.Log("Put objects to bucket")

	f, err := os.Open(m.addonSource)
	if err != nil {
		return err
	}
	defer f.Close()

	uncompressedStream, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(uncompressedStream)
	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			path := filepath.Join(m.minioTmp, header.Name)
			if err := os.Mkdir(path, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			path := filepath.Join(m.minioTmp, header.Name)
			outFile, err := os.Create(path)
			if err != nil {
				return err
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}

			_, err = m.client.FPutObject(
				bucketName,
				fmt.Sprintf("%s/%s", bucketDir, strings.TrimLeft(header.Name, "./")),
				path,
				minio.PutObjectOptions{})
			if err != nil {
				return errors.Wrap(err, "while putting objects to bucket")
			}
		default:
			m.t.Logf("ExtractTarGz: uknown type: %v in %s", header.Typeflag, header.Name)
		}
	}

	m.t.Log("Set bucket policy")
	policy := `{"Version": "2012-10-17","Statement": [{"Action": ["s3:GetObject"],"Effect": "Allow","Principal": {"AWS": ["*"]},"Resource": ["arn:aws:s3:::addon-test/*"],"Sid": "AddPerm"}]}`
	err = m.client.SetBucketPolicy(bucketName, policy)
	if err != nil {
		return errors.Wrap(err, "while setting policy for bucket")
	}

	return nil
}

func (m *minioServer) minioURL(index string) string {
	return fmt.Sprintf(
		"http://%s:%s/%s/%s//addons/%s?aws_access_key_id=%s&aws_access_key_secret=%s&region=%s",
		minioIP,
		minioPort,
		bucketName,
		bucketDir,
		index,
		accessKey,
		accessSecret,
		bucketRegion)
}
