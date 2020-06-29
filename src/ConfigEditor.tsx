import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { DataSourceOptions, SecureJsonData } from './types';

const { SecretFormField, FormField } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<DataSourceOptions> {}

interface State {}

export class ConfigEditor extends PureComponent<Props, State> {
  onOptionChange = (what: string) => (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData: any = {
      ...options.jsonData,
    };
    jsonData[what] = event.target.value
    onOptionsChange({ ...options, jsonData });
  };

  // Secure field (only sent to the backend)
  onSecureOptionChange = (what: string) => (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const secureJsonData: any = {};
    secureJsonData[what] = event.target.value
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
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as SecureJsonData;

    return (
      <div className="gf-form-group">
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
            inputWidth={20}
            onChange={this.onOptionChange('region')}
            value={jsonData.region || ''}
            placeholder="Region"
            required
          />
        </div>

        <div className="gf-form-inline">
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
        </div>

        <div className="gf-form-inline">
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
        </div>
      </div>
    );
  }
}
