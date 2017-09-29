# Ingest Web Console

## Environment Set-up
1. Install virtualenv, create an environment, and activate the newly created environment.
2. Run `pip install -t lib -r requirements.txt` to install the requirements for the backend Flask application.

## Local Deployment
From `cloud-ingest/webconsole/back-end` run
`env INGEST_CONFIG_PATH=ingestwebconsole.local_settings python main.py`

## Testing
From `cloud-ingest/webconsole/back-end` run `python -m unittest discover`.

## App Engine Deployment
1. Make sure you have installed the [Google Cloud SDK](https://cloud.google.com/sdk/docs/)
2. Run `gcloud app deploy --project <your-cloud-project-id> app.<test|prod>.yaml`
   with the id of your Google Cloud project.
