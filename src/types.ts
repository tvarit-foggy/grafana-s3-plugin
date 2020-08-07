import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface Query extends DataQuery {
  path: string;
  compression: string;
  format: string;
  query: string;
  csv_allow_quoted_record_delimiter: boolean;
  csv_comments: string;
  csv_field_delimiter: string;
  csv_file_header_info: string;
  csv_quote_character: string;
  csv_quote_escape_character: string;
  csv_record_delimiter: string;
  json_type: string;
  json_time_field_enable: boolean;
  json_time_field: string;
  json_time_month_first: boolean;
  json_time_bucket: number;
}

export const defaultQuery: Partial<Query> = {
  compression: 'NONE',
  format: 'CSV',
  query: 'SELECT * FROM S3Object',
  csv_allow_quoted_record_delimiter: false,
  csv_comments: '#',
  csv_field_delimiter: ',',
  csv_file_header_info: 'USE',
  csv_quote_character: '"',
  csv_quote_escape_character: '"',
  csv_record_delimiter: '\n',
  json_type: 'DOCUMENT',
  json_time_field_enable: false,
  json_time_field: '',
  json_time_month_first: false,
  json_time_bucket: 1000000,
};

/**
 * These are options configured for each DataSource instance
 */
export interface DataSourceOptions extends DataSourceJsonData {
  bucket: string;
  accessKey?: string;
  region: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface SecureJsonData {
  secretKey?: string;
}

/**
 * Select Options
 */
export const CompressionTypes = [
  { label: 'NONE', value: 'NONE' },
  { label: 'GZIP', value: 'GZIP' },
  { label: 'BZIP2', value: 'BZIP2' },
];

export const FileFormats = [
  { label: 'CSV', value: 'CSV' },
  { label: 'JSON', value: 'JSON' },
  { label: 'PARQUET', value: 'PARQUET' },
];

export const JsonTypes = [
  { label: 'DOCUMENT', value: 'DOCUMENT' },
  { label: 'LINES', value: 'LINES' },
];

export const CsvFileHeaderInfo = [
  { label: 'NONE', value: 'NONE', description: 'First line is not a header.' },
  {
    label: 'IGNORE',
    value: 'IGNORE',
    description:
      'First line is a header, but you cant use the header values to indicate the column in an expression. You can use column position (such as _1, _2, …) to indicate the column (SELECT s._1 FROM OBJECT s).',
  },
  {
    label: 'USE',
    value: 'USE',
    description:
      'First line is a header, and you can use the header value to identify a column in an expression (SELECT "name" FROM OBJECT).',
  },
];

export const Regions = [
  { label: 'Europe (Frankfurt)', value: 'eu-central-1' },
  { label: 'Europe (Ireland)', value: 'eu-west-1' },
  { label: 'Europe (London)', value: 'eu-west-2' },
  { label: 'Europe (Milan)', value: 'eu-south-1' },
  { label: 'Europe (Paris)', value: 'eu-west-3' },
  { label: 'Europe (Stockholm)', value: 'eu-north-1' },
  { label: 'US East (N. Virginia)', value: 'us-east-1' },
  { label: 'US East (Ohio)', value: 'us-east-2' },
  { label: 'US West (N. California)', value: 'us-west-1' },
  { label: 'US West (Oregon)', value: 'us-west-2' },
  { label: 'Africa (Cape Town)', value: 'af-south-1' },
  { label: 'Asia Pacific (Hong Kong)', value: 'ap-east-1' },
  { label: 'Asia Pacific (Mumbai)', value: 'ap-south-1' },
  { label: 'Asia Pacific (Seoul)', value: 'ap-northeast-2' },
  { label: 'Asia Pacific (Singapore)', value: 'ap-southeast-1' },
  { label: 'Asia Pacific (Sydney)', value: 'ap-southeast-2' },
  { label: 'Asia Pacific (Tokyo)', value: 'ap-northeast-1' },
  { label: 'Canada (Central)', value: 'ca-central-1' },
  { label: 'Middle East (Bahrain)', value: 'me-south-1' },
  { label: 'South America (São Paulo)', value: 'sa-east-1' },
];
