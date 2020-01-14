## Providing Custom Test Values

In order to enable custom test values, add any number of *-values.yaml files to this directory. Only files with a suffix -values.yaml are considered. Instead of using the defaults, the chart is then installed and tested separately for each of these files using the --values flag.

Please note that in order to test using the default values when using the ci directory, an empty values file must be present in the directory.