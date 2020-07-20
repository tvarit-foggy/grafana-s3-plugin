# AWS S3 Plugin

This plugin queries files on AWS S3 using S3 Select API

### Installation
To install latest stable version:
```
./bin/grafana-cli --pluginUrl https://github.com/tvarit-foggy/grafana-s3-plugin/releases/download/v0.0.1/tvarit-aws-s3.zip plugins install tvarit-s3-datasource
```
To install git master, download the repository as zip file
```
./bin/grafana-cli --pluginUrl <path_to_zip_file> plugins install tvarit-s3-datasource
```

### Screenshots
![Config Editor](src/img/config.png?raw=true "Config Editor")
![Query Editor](src/img/query.png?raw=true "Query Editor")

### Links
S3 Select API Reference: https://docs.aws.amazon.com/AmazonS3/latest/API/API_SelectObjectContent.html  
Query Reference: https://docs.aws.amazon.com/AmazonS3/latest/dev/s3-glacier-select-sql-reference.html
