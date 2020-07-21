import defaults from 'lodash/defaults';

import React, { ChangeEvent, PureComponent } from 'react';
import { css } from 'emotion';
import { LegacyForms } from '@grafana/ui';
import { QueryEditorProps, SelectableValue } from '@grafana/data';
import { DataSource } from './DataSource';
import { defaultQuery, DataSourceOptions, Query } from './types';
import { CompressionTypes, FileFormats, JsonTypes, CsvFileHeaderInfo } from './types';

const { FormField, Select, Switch } = LegacyForms;

type Props = QueryEditorProps<DataSource, Query, DataSourceOptions>;

interface State {
  compression: SelectableValue<string>;
  format: SelectableValue<string>;
  json_type: SelectableValue<string>;
  csv_file_header_info: SelectableValue<string>;
}

export class QueryEditor extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props);

    const { compression, format, json_type, csv_file_header_info } = defaults(this.props.query, defaultQuery);

    this.state = {
      compression: { label: compression, value: compression },
      format: { label: format, value: format },
      json_type: { label: json_type, value: json_type },
      csv_file_header_info: { label: csv_file_header_info, value: csv_file_header_info },
    };
  }

  onInputChange = (what: string) => (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;

    const _query: any = {
      ...query,
    };
    _query[what] = event.target.value;
    onChange(_query);

    // executes the query
    onRunQuery();
  };

  onIntInputChange = (what: string) => (event: ChangeEvent<HTMLInputElement>) => {
    const { onChange, query, onRunQuery } = this.props;

    const _query: any = {
      ...query,
    };
    _query[what] = parseInt(event.target.value, 10);
    onChange(_query);

    // executes the query
    onRunQuery();
  };

  onSelectChange = (what: string) => (value: SelectableValue<string>) => {
    const { onChange, query, onRunQuery } = this.props;

    const _query: any = {
      ...query,
    };
    _query[what] = value.value;
    onChange(_query);

    const _state: any = {};
    _state[what] = value;
    this.setState(_state);

    // executes the query
    onRunQuery();
  };

  onSwitchChange = (what: string) => () => {
    const { onChange, query, onRunQuery } = this.props;

    const _query: any = {
      ...query,
    };
    _query[what] = !_query[what];

    if (what === 'json_time_field_enable' && !_query[what]) {
      _query['json_time_field'] = '';
    }

    onChange(_query);
    onRunQuery();
  };

  render() {
    const {
      path,
      query,
      csv_allow_quoted_record_delimiter,
      csv_comments,
      csv_field_delimiter,
      csv_quote_character,
      csv_quote_escape_character,
      csv_record_delimiter,
      json_time_field,
      json_time_month_first,
      json_time_bucket,
    } = defaults(this.props.query, defaultQuery);

    const { compression, format, json_type, csv_file_header_info } = this.state;

    const sectionHeader = css`
      margin: 8px 0px 4px 8px;
      font-weight: 500;
    `;

    let json_time_field_enable = this.props.query.json_time_field_enable || !!json_time_field;

    return (
      <>
        <div className="gf-form" style={{ flexWrap: 'wrap' }}>
          <FormField
            labelWidth={10}
            inputWidth={0}
            className={css`
              flex-grow: 1;
            `}
            value={path}
            onChange={this.onInputChange('path')}
            label="Path"
            tooltip="Path of object that is queried."
          />
          <FormField
            labelWidth={10}
            label="Format"
            tooltip="Describes the format of the data in the object that is being queried."
            inputEl={
              <Select width={10} options={FileFormats} value={format} onChange={this.onSelectChange('format')} />
            }
          />
          <FormField
            labelWidth={10}
            label="Compression"
            tooltip="Specifies object's compression format."
            inputEl={
              <Select
                width={9}
                options={CompressionTypes}
                value={compression}
                onChange={this.onSelectChange('compression')}
              />
            }
          />
        </div>
        <div className="gf-form gf-form--grow">
          <FormField
            labelWidth={10}
            inputWidth={0}
            className={css`
              flex-grow: 1;
            `}
            value={query}
            onChange={this.onInputChange('query')}
            label="Query"
            tooltip="The expression that is used to query the object."
          />
        </div>
        {format.value === 'CSV' && (
          <>
            <div className={sectionHeader}>CSV Details</div>
            <div className="gf-form" style={{ flexWrap: 'wrap' }}>
              <FormField
                labelWidth={15}
                inputWidth={5}
                value={csv_field_delimiter}
                onChange={this.onInputChange('csv_field_delimiter')}
                label="Field Delimiter"
                tooltip="A single character used to separate individual fields in a record. You can specify an arbitrary delimiter."
              />
              <FormField
                labelWidth={15}
                label="File Header Info"
                tooltip="Describes the first line of input."
                inputEl={
                  <Select
                    width={5}
                    options={CsvFileHeaderInfo}
                    value={csv_file_header_info}
                    onChange={this.onSelectChange('csv_file_header_info')}
                  />
                }
              />
              <FormField
                labelWidth={15}
                inputWidth={5}
                value={csv_quote_character}
                onChange={this.onInputChange('csv_quote_character')}
                label="Quote Character"
                tooltip='A single character used for escaping when the field delimiter is part of the value. For example, if the value is a, b, Amazon S3 wraps this field value in quotation marks, as follows: " a , b ".'
              />
              <FormField
                labelWidth={15}
                inputWidth={5}
                value={csv_quote_escape_character}
                onChange={this.onInputChange('csv_quote_escape_character')}
                label="Quote Escape Character"
                tooltip='A single character used for escaping the quotation mark character inside an already escaped value. For example, the value """ a , b """ is parsed as " a , b ".'
              />
              <FormField
                labelWidth={15}
                inputWidth={5}
                value={csv_comments}
                onChange={this.onInputChange('csv_comments')}
                label="Comments"
                tooltip="A single character used to indicate that a row should be ignored when the character is present at the start of that row. You can specify any character to indicate a comment line."
              />
              <FormField
                labelWidth={15}
                inputWidth={5}
                value={csv_record_delimiter}
                onChange={this.onInputChange('csv_record_delimiter')}
                label="Record Delimiter"
                tooltip="A single character used to separate individual records in the input. Instead of the default value, you can specify an arbitrary delimiter."
              />
              <FormField
                labelWidth={15}
                label="Allow Quoted Record Delimiter"
                tooltip="Specifies that CSV field values may contain quoted record delimiters and such records should be allowed. Default value is FALSE. Setting this value to TRUE may lower performance."
                inputEl={
                  <Switch
                    label=""
                    checked={csv_allow_quoted_record_delimiter}
                    onChange={this.onSwitchChange('csv_allow_quoted_record_delimiter')}
                  />
                }
              />
            </div>
          </>
        )}
        {format.value === 'JSON' && (
          <>
            <div className={sectionHeader}>JSON Details</div>
            <div className="gf-form">
              <FormField
                labelWidth={10}
                label="Type"
                tooltip="The type of JSON."
                inputEl={
                  <Select
                    width={10}
                    options={JsonTypes}
                    value={json_type}
                    onChange={this.onSelectChange('json_type')}
                  />
                }
              />
              <FormField
                labelWidth={10}
                label="Has Time Field?"
                tooltip="Does the JSON have a field with packet timestamp"
                inputEl={
                  <Switch
                    label=""
                    checked={json_time_field_enable}
                    onChange={this.onSwitchChange('json_time_field_enable')}
                  />
                }
              />
            </div>
            {json_time_field_enable && (
              <div className="gf-form">
                <FormField
                  labelWidth={10}
                  inputWidth={0}
                  className={css`
                    flex-grow: 1;
                  `}
                  value={json_time_field}
                  onChange={this.onInputChange('json_time_field')}
                  label="Time Field"
                  tooltip="The expression that is used to query time field."
                />
                <FormField
                  labelWidth={10}
                  label="Month First"
                  tooltip="Is time field month first?"
                  inputEl={
                    <Switch
                      label=""
                      checked={json_time_month_first}
                      onChange={this.onSwitchChange('json_time_month_first')}
                    />
                  }
                />
                <FormField
                  labelWidth={10}
                  inputWidth={10}
                  type="number"
                  value={json_time_bucket}
                  onChange={this.onIntInputChange('json_time_bucket')}
                  label="Time Bucket"
                  tooltip="Time difference between each entry within a packet in nanoseconds"
                />
              </div>
            )}
          </>
        )}
      </>
    );
  }
}
