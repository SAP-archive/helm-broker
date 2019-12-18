package helm

// WithHelmClientFactory creates HelmClient with specified DeleteInstaller factory
func (cli *Client) WithHelmClientFactory(factory func() DeleteInstaller) *Client {
	cli.helmClientFactory = factory
	return cli
}
