import React, { FC, useState } from 'react';
import { css } from 'emotion';
import { getBackendSrv } from '@grafana/runtime';
import { stylesFactory, useTheme, getFormStyles, Icon } from '@grafana/ui';
import { base64StringToArrowTable, GrafanaTheme } from '@grafana/data';
import * as AWS from 'aws-sdk';
import * as $ from 'jquery';

import LinearProgress from '@material-ui/core/LinearProgress';
import Typography from '@material-ui/core/Typography';
import Box from '@material-ui/core/Box';

type ComponentSize = 'xs' | 'sm' | 'md' | 'lg';

interface Props {
  orgId: number;
  dsId: number;
  bucket: string;
  region: string;
  prefix: string;
  refresh: () => void;
}

interface Progress {
  busy: boolean;
  message: string;
  progress: number;
}

export const FileUploader: FC<Props> = props => {
  const [errors, _setErrors] = useState<string[]>([]);

  const setError = (error: string) => {
    errors.push(error);
    _setErrors(errors);
  };

  const [progress, _setProgress] = useState<Progress>({
    busy: false,
    message: 'Done',
    progress: 100,
  });

  const setProgress = (progress: Progress) => {
    console.log(progress.message, progress.progress, '%');
    _setProgress(progress);
  };

  let credentials: any = undefined;
  const authenticate = () => {
    if (credentials && !credentials.expired) {
      return Promise.resolve();
    }

    setProgress({
      busy: true,
      message: 'Authenticating',
      progress: 0,
    });

    return getBackendSrv()
      .datasourceRequest({
        url: '/api/ds/query',
        method: 'POST',
        data: {
          queries: [
            {
              refId: 'A',
              query: 'UPLOAD',
              orgId: props.orgId,
              datasourceId: props.dsId,
            },
          ],
        },
      })
      .then(response => {
        setProgress({
          busy: true,
          message: 'Authenticating',
          progress: 80,
        });
        const b64 = response.data.results.A.dataframes[0];
        const table = base64StringToArrowTable(b64);
        credentials = new AWS.Credentials({
          accessKeyId: table.getColumn('AccessKeyId').get(0),
          secretAccessKey: table.getColumn('SecretAccessKey').get(0),
          sessionToken: table.getColumn('SessionToken').get(0),
        });
        AWS.config.update({
          region: props.region,
          credentials: credentials,
        });
        setProgress({
          busy: true,
          message: 'Authenticating',
          progress: 100,
        });
      });
  };

  const uploadFile = (file: any, i: number, n: number) => {
    return authenticate().then(() => {
      setProgress({
        busy: true,
        message: 'Uploading (' + i + ' of ' + n + '): "' + file.name + '"',
        progress: 0,
      });
      return new AWS.S3.ManagedUpload({
        params: {
          Bucket: props.bucket,
          Key: props.prefix + file.name,
          Body: file,
          ACL: 'private',
        },
      })
        .on('httpUploadProgress', function(evt) {
          setProgress({
            busy: true,
            message: 'Uploading (' + i + ' of ' + n + '): "' + file.name + '"',
            progress: (100 * evt.loaded) / evt.total,
          });
        })
        .promise()
        .catch((err: any) => {
          setProgress({
            busy: true,
            message: file.name + ': ' + err.message,
            progress: 100,
          });
          setError(file.name + ': ' + err.message);
        });
    });
  };

  const onFileUpload = async (event: any) => {
    const files = event.target.files;

    if (!files.length) {
      return;
    }

    _setErrors([]);
    for (let i = 0; i < files.length; i++) {
      await uploadFile(files[i], i + 1, files.length);
      props.refresh();
    }

    setProgress({
      busy: false,
      message: 'Done',
      progress: 100,
    });
    // @ts-ignore
    $('#uploadInput').val('');
  };

  const theme = useTheme();
  const style = getStyles(theme, 'md');

  return (
    <div style={{ width: '100%' }}>
      <Box display="flex" alignItems="center" mb="12px">
        <Box minWidth={35} key="button">
          {/*
          // @ts-ignore*/}
          <label className={style.button} disabled={progress.busy}>
            <Icon name="upload" className={style.icon} />
            Upload files
            <input
              type="file"
              id="uploadInput"
              className={style.fileUpload}
              onChange={onFileUpload}
              multiple={true}
              disabled={progress.busy}
            />
          </label>
        </Box>
        {progress.busy && [
          <Box width="calc(100% - 200px)" mx="12px" key="progress">
            <Typography variant="body2" noWrap={true} color="textSecondary">
              {progress.message}
            </Typography>
            <LinearProgress variant="determinate" color="secondary" value={progress.progress} />
          </Box>,
          <Box width="35px" key="label">
            <Typography variant="body2" color="textSecondary">
              &nbsp;
            </Typography>
            <Typography variant="body2" color="textSecondary">{`${Math.round(progress.progress)}%`}</Typography>
          </Box>,
        ]}
      </Box>
      {errors.map(err => (
        <Typography variant="body2" noWrap={true} color="error">
          {err}
        </Typography>
      ))}
    </div>
  );
};

const getStyles = stylesFactory((theme: GrafanaTheme, size: ComponentSize) => {
  // @ts-ignore
  const buttonFormStyle = getFormStyles(theme, { invalid: false, size }).button.button;
  return {
    fileUpload: css`
      display: none;
    `,
    button: css`
      ${buttonFormStyle}
    `,
    icon: css`
      margin-right: ${theme.spacing.xs};
    `,
  };
});
