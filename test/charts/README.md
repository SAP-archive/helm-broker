# Charts Tests

This directory contains tests that validate if the Helm Broker [charts](../charts) work as expected after installation. These tests also help to understand the purpose of a given chart.

## Details

Find the definition of a Pod that runs tests in the `templates/tests` directory. The Pod definition contains the Helm `helm.sh/hook: test-success` hook annotation, which means that the Pod is executed only when you run the `helm test {relase_name}` command.
