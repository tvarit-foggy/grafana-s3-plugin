# AWS S3 Plugin [![tvarit-foggy](https://circleci.com/gh/tvarit-foggy/grafana-s3-plugin.svg?style=svg)](https://app.circleci.com/pipelines/github/tvarit-foggy/grafana-s3-plugin)

This plugin queries files on AWS S3 using S3 Select API

### Installation
To install, download zip file from release page (for stable version) or download repository as zip file (for installing from git master)
```
grafana-cli --pluginUrl <path_to_zip_file> plugins install tvarit-s3-datasource
```

The plugin is not signed. To enable this plugin, change the following

```
[plugins]
...
# Enter a comma-separated list of plugin identifiers to identify plugins that are allowed to be loaded even if they lack a valid signature.
allow_loading_unsigned_plugins =
```
to

```
[plugins]
...
# Enter a comma-separated list of plugin identifiers to identify plugins that are allowed to be loaded even if they lack a valid signature.
allow_loading_unsigned_plugins = tvarit-s3-datasource
```

### Screenshots
![Config Editor](src/img/config.png?raw=true "Config Editor")
![Query Editor](src/img/query.png?raw=true "Query Editor")

### Links
S3 Select API Reference: https://docs.aws.amazon.com/AmazonS3/latest/API/API_SelectObjectContent.html  
Query Reference: https://docs.aws.amazon.com/AmazonS3/latest/dev/s3-glacier-select-sql-reference.html
