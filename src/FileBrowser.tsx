import React, { PureComponent } from 'react';
import { Button, Table, Select, Drawer } from '@grafana/ui';
import { getDisplayProcessor, DataFrame, base64StringToArrowTable, arrowTableToDataFrame } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import Breadcrumbs from '@material-ui/core/Breadcrumbs';
import { FileUploader } from './FileUploader';

interface Props {
  orgId: number;
  dsId: number;
  bucket: string;
  region: string;
}

interface State {
  explore: boolean;
  prefix: string;
  dirty: boolean;
  files?: DataFrame;
  hints: any;
}

export class FileBrowser extends PureComponent<Props, State> {
  state: State = {
    explore: false,
    prefix: '',
    dirty: true,
    hints: [],
  };

  icons: any = {
    folder: '🗀 ',
    file: '🖹 ',
    error: '⚠ ',
    delete: '🗑 ',
  };

  constructor(props: Props) {
    super(props);
    const params = this.getsearchparams();
    this.state.explore = params['explore'];
    this.state.prefix = params['prefix'];
    if (this.state.explore) {
      this.listFiles(this.state.prefix);
    }

    window.onpopstate = (e: any) => {
      this.getsearchparams(true);
    };
  }

  getsearchparams = (event?: any) => {
    const params: any = {};
    if (window.location.search) {
      window.location.search
        .substr(1)
        .split('&')
        .forEach(part => {
          const item = part.split('=');
          params[item[0]] = decodeURIComponent(item[1]);
        });
    }
    params['prefix'] = params['prefix'] || '';
    params['explore'] = params['explore'] === 'true';

    if (event) {
      if (this.state.explore !== params['explore'] || this.state.prefix !== params['prefix']) {
        console.log('navigating to', params['prefix']);
        this.setState({
          explore: params['explore'],
          prefix: params['prefix'],
          dirty: true,
          hints: [],
        });
        if (params['explore']) {
          this.listFiles(params['prefix']);
        }
      }
    }

    return params;
  };

  setsearchparams = (new_params: any, update?: boolean) => {
    const params = this.getsearchparams();
    for (const k in new_params) {
      params[k] = new_params[k];
    }
    let search: any = [];
    for (const k in params) {
      search.push(k + '=' + encodeURIComponent(params[k]));
    }

    search = '?' + search.join('&');
    const url = window.location.pathname + search;

    if (window.location.search !== search && update) {
      window.history.pushState({ prefix: params['prefix'] || '' }, 'S3 File Explorer', window.location.origin + url);
    }

    return url;
  };

  refresh() {
    this.listFiles(this.state.prefix);
  }

  listFiles = (prefix: string, edit?: boolean) => {
    getBackendSrv()
      .datasourceRequest({
        url: '/api/ds/query',
        method: 'POST',
        data: {
          queries: [
            {
              refId: 'A',
              path: prefix,
              query: 'LIST FORMATTED',
              orgId: this.props.orgId,
              datasourceId: this.props.dsId,
            },
          ],
        },
      })
      .then(response => {
        const b64 = response.data.results.A.dataframes[0];
        const table = base64StringToArrowTable(b64);
        const frame = arrowTableToDataFrame(table);

        frame.fields[0].display = (value: string) => {
          let icon = this.icons['error'];

          const parts = /^(.*),type=(.*),key=(.*)$/.exec(value);
          if (parts !== null && parts.length === 4) {
            value = parts![1];
            icon = this.icons[parts![2]] || icon;
          }

          return {
            prefix: icon,
            numeric: Number.NaN,
            text: value,
          };
        };

        // @ts-ignore
        frame.fields[0].getLinks = (options: any) => {
          const i = options.valueRowIndex;

          let value = table.getColumnAt(0)!.get(i);
          let href = undefined;

          const parts = /^(.*),type=(.*),key=(.*)$/.exec(value);
          if (parts !== null && parts.length === 4 && parts![2] === 'folder') {
            value = parts![1];
            href = this.setsearchparams({ explore: true, prefix: parts![3] });
          }

          if (href) {
            return [{ href: href, title: value }];
          } else {
            return [undefined];
          }
        };

        frame.fields[1].display = getDisplayProcessor({ field: frame.fields[1] });
        frame.fields[2].display = getDisplayProcessor({ field: frame.fields[2] });

        frame.fields[3].display = (value: string) => {
          return {
            numeric: Number.NaN,
            text: this.icons['delete'],
          };
        };

        // @ts-ignore
        frame.fields[3].getLinks = (options: any) => {
          const i = options.valueRowIndex;
          const value = table.getColumnAt(0)!.get(i);
          let onClick = undefined;

          const parts = /^(.*),type=(.*),key=(.*)$/.exec(value);
          if (parts !== null && parts.length === 4) {
            onClick = () => {
              console.log('Delete', parts![2], parts![3]);
              return getBackendSrv()
                .datasourceRequest({
                  url: '/api/ds/query',
                  method: 'POST',
                  data: {
                    queries: [
                      {
                        refId: 'A',
                        path: parts![3],
                        query: 'DELETE ' + parts![2],
                        orgId: this.props.orgId,
                        datasourceId: this.props.dsId,
                      },
                    ],
                  },
                })
                .then(response => {
                  this.listFiles(this.state.prefix);
                });
            };
          }

          return [{ onClick: onClick, title: 'Delete' }];

          if (onClick) {
            return [{ onClick: onClick, title: 'Delete' }];
          } else {
            return [undefined];
          }
        };

        if (edit) {
          this.setState({
            hints: frame!.meta!.custom!.folders.map((folder: any) => ({ label: folder, value: folder })),
          });
        } else if (prefix === this.state.prefix) {
          this.setState({
            dirty: false,
            files: frame,
            hints: frame!.meta!.custom!.folders.map((folder: any) => ({ label: folder, value: folder })),
          });
        }
      });
  };

  getBase = (prefix: string) => {
    const i = prefix.lastIndexOf('/') + 1;
    return prefix.slice(0, i);
  };

  onBreadcrumbClick = (event: any) => {
    event.preventDefault ? event.preventDefault() : 0;
    event.stopPropagation ? event.stopPropagation() : 0;

    const prefix = event.target.getAttribute('data-prefix');
    if (prefix !== this.state.prefix) {
      console.log('navigating to', prefix);
      this.setState({ prefix: prefix, dirty: true, hints: [] });
      this.setsearchparams({ prefix: prefix }, true);
      this.listFiles(prefix);
    } else if (prefix !== '' && prefix.endsWith('/')) {
      console.log('switching to edit mode');
      const base = this.getBase(prefix.slice(0, prefix.length - 1));
      this.setState({ prefix: prefix.slice(0, prefix.length - 1) });
      this.listFiles(base, true);
    }
  };

  onBreadcrumbChange = (option: any) => {
    if (option.value && !option.value.endsWith('/')) {
      option.value = option.value + '/';
    }
    const prefix = this.getBase(this.state.prefix) + option.value;
    console.log('navigating to', prefix);
    this.setState({ prefix: prefix, dirty: true, hints: [] });
    this.listFiles(prefix);
  };

  onDrawerOpen = () => {
    this.setState({ explore: true });
    this.setsearchparams({ explore: true, prefix: this.state.prefix }, true);
    this.listFiles(this.state.prefix);
  };

  onDrawerClose = () => {
    this.setState({ explore: false });
    this.setsearchparams({ explore: false, prefix: this.state.prefix }, true);
  };

  render() {
    const { orgId, dsId, bucket, region } = this.props;
    const { explore, prefix, dirty, files, hints } = this.state;

    const crumbs = this.state.prefix.split('/').map((part, i) => {
      let value = this.state.prefix
        .split('/')
        .slice(0, i + 1)
        .join('/');
      if (value !== '') {
        value = value + '/';
      }
      return {
        label: part,
        value: value,
      };
    });

    crumbs.unshift({
      label: 'Root',
      value: '',
    });

    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <Button variant="link" onClick={this.onDrawerOpen}>
            Click here to open S3 File Explorer
          </Button>
        </div>
        {explore && (
          <Drawer
            title="File Explorer"
            subtitle={
              <Breadcrumbs maxItems={4} aria-label="breadcrumb">
                {crumbs.slice(0, crumbs.length - 1).map(part => {
                  return (
                    <a href="#" data-prefix={part.value} onClick={this.onBreadcrumbClick} key={part.value}>
                      {part.label}
                    </a>
                  );
                })}
                <Select
                  allowCustomValue={true}
                  closeMenuOnSelect={true}
                  isSearchable={true}
                  tabSelectsValue={true}
                  onChange={this.onBreadcrumbChange}
                  options={hints}
                  value={crumbs[crumbs.length - 1]}
                  width={20}
                />
              </Breadcrumbs>
            }
            onClose={this.onDrawerClose}
            expandable={true}
            scrollableContent={true}
            width={720}
          >
            <div className="gf-form-group">
              <div className="gf-form">
                <FileUploader
                  orgId={orgId}
                  dsId={dsId}
                  bucket={bucket}
                  region={region}
                  prefix={prefix}
                  refresh={this.refresh.bind(this)}
                />
              </div>
            </div>
            <div className="gf-form-group">
              {files && !dirty && (
                <div className="gf-form" onClick={this.getsearchparams}>
                  <Table height={540} width={688} data={files} resizable={true} />
                </div>
              )}
            </div>
          </Drawer>
        )}
      </div>
    );
  }
}
