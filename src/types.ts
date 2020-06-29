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
}

export const defaultQuery: Partial<Query> = {
  compression: "NONE",
  format: "CSV",
  query: "SELECT * FROM S3Object",
  csv_allow_quoted_record_delimiter: false,
  csv_comments: "#",
  csv_field_delimiter: ",",
  csv_file_header_info: "USE",
  csv_quote_character: "\"",
  csv_quote_escape_character: "\"",
  csv_record_delimiter: "\n",
  json_type: "DOCUMENT",
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
  { label: "NONE", value: "NONE" },
  { label: "GZIP", value: "GZIP" },
  { label: "BZIP2", value: "BZIP2" },
];

export const FileFormats = [
  { label: "CSV", value: "CSV" },
  { label: "JSON", value: "JSON" },
  { label: "PARQUET", value: "PARQUET" },
];

export const JsonTypes = [
  { label: "DOCUMENT", value: "DOCUMENT" },
  { label: "LINES", value: "LINES" },
];

export const CsvFileHeaderInfo = [
  { label: "NONE", value: "NONE", description: "First line is not a header." },
  { label: "IGNORE", value: "IGNORE", description: "First line is a header, but you can't use the header values to indicate the column in an expression. You can use column position (such as _1, _2, …) to indicate the column (SELECT s._1 FROM OBJECT s)." },
  { label: "USE", value: "USE", description: "First line is a header, and you can use the header value to identify a column in an expression (SELECT \"name\" FROM OBJECT)." },
];
