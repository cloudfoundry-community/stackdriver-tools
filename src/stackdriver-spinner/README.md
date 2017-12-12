# Log Spinner CF App

## Instructions to run stackdriver-spinner
1. Add the target GCP project in `stackdriver-tools/src/stackdriver-spinner/manifest.yml`.
1. The GCP project will need a service account with log reader and writer permissions.
1. The JSON key for this service account should be stored in `stackdriver-tools/src/stackdriver-spinner/credentials.json`.
1. Make sure that you are logged into cf and have targeted the desired org and space.
1. Run the following: `cd stackdriver-tools/src/stackdriver-spinner & cf push`.
1. Search for "Loss: " in the stackdriver logging platform. You should be able to see the loss measurement on a scale from 0 to 1 (1 representing complete loss of logs).