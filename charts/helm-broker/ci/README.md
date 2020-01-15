## Providing custom test values

In order to enable custom test values, add any number of `-values.yaml` files to this directory. Only files with the `-values.yaml` suffix are considered. Instead of using the defaults, the chart is installed and tested separately for each of these files using the `--values` flag.

If you want to perform testing using the default values, an empty `values.yaml` file must be present in the `ci` directory.
