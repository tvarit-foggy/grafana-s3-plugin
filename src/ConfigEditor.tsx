import _ from 'lodash';
import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps, SelectableValue } from '@grafana/data';
import { DataSourceOptions, SecureJsonData, Regions } from './types';
import { FileBrowser } from './FileBrowser';

const { SecretFormField, FormField, Select } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<DataSourceOptions> {}

interface State {
  region: SelectableValue<string>;
}

export class ConfigEditor extends PureComponent<Props, State> {
  constructor(props: Props) {
    super(props);
    const region = _.find(Regions, { value: props.options.jsonData.region }) || Regions[0];
    this.state = { region: region };
  }

  onOptionChange = (what: string) => (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData: any = {
      ...options.jsonData,
    };
    jsonData[what] = event.target.value;
    onOptionsChange({ ...options, jsonData });
  };

  onSelectChange = (what: string) => (value: SelectableValue<string>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData: any = {
      ...options.jsonData,
    };
    jsonData[what] = value.value;
    onOptionsChange({ ...options, jsonData });

    const _state: any = {};
    _state[what] = value;
    this.setState(_state);
  };

  // Secure field (only sent to the backend)
  onSecureOptionChange = (what: string) => (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const secureJsonData: any = {};
    secureJsonData[what] = event.target.value;
    onOptionsChange({
      ...options,
      secureJsonData,
    });
  };

  onResetSecureOption = (what: string) => () => {
    const { onOptionsChange, options } = this.props;
    const secureJsonFields: any = {
      ...options.secureJsonFields,
    };
    secureJsonFields[what] = false;
    const secureJsonData: any = {
      ...options.secureJsonData,
    };
    secureJsonData[what] = '';
    onOptionsChange({
      ...options,
      secureJsonFields,
      secureJsonData,
    });
  };

  render() {
    const { region } = this.state;
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as SecureJsonData;

    return [
      <div className="gf-form-group" key="configure">
        <div className="gf-form">
          <FormField
            label="Bucket"
            labelWidth={10}
            inputWidth={20}
            onChange={this.onOptionChange('bucket')}
            value={jsonData.bucket || ''}
            placeholder="Bucket"
            required
          />
        </div>
        <div className="gf-form">
          <FormField
            label="Region"
            labelWidth={10}
            required
            inputEl={<Select width={20} options={Regions} value={region} onChange={this.onSelectChange('region')} />}
          />
        </div>
        <div className="gf-form">
          <FormField
            label="Access Key ID"
            labelWidth={10}
            inputWidth={20}
            onChange={this.onOptionChange('accessKey')}
            value={jsonData.accessKey || ''}
            placeholder="Access Key ID"
          />
        </div>
        <div className="gf-form">
          <SecretFormField
            isConfigured={(secureJsonFields && secureJsonFields.secretKey) as boolean}
            value={secureJsonData.secretKey || ''}
            label="Secret Access Key"
            placeholder="Secret Access Key"
            labelWidth={10}
            inputWidth={20}
            onReset={this.onResetSecureOption('secretKey')}
            onChange={this.onSecureOptionChange('secretKey')}
          />
        </div>
      </div>,
      <FileBrowser
        key="explore"
        orgId={options.orgId}
        dsId={options.id}
        bucket={jsonData.bucket || ''}
        region={jsonData.region || Regions[0].value}
      />,
    ];
  }
}
