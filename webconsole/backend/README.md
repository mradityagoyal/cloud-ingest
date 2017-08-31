# Ingest Web Console

## Environment Set-up
1. Install virtualenv, create an environment, and activate the newly created environment.
2. Run `pip install -t lib -r requirements.txt` to install the requirements for the backend Flask application.

## Local Deployment
1. Create a settings file (such as ingestwebconsole.local_settings) and
   store the following values in it:
   ```
   SPANNER_INSTANCE='<the id of your Cloud Spanner instance>'
   SPANNER_DATABASE='<the name of the Spanner database>'
   ```

   Here is an example file:
   ```
   SPANNER_INSTANCE='cloud-ingest-spanner-instance'
   SPANNER_DATABASE='cloud-ingest-database'
   ```
2. Set the `INGEST_CONFIG_PATH` environment variable to hold the path to
   your local config file.
Linux: `export INGEST_CONFIG_PATH="example/path/to/ingestwebconsole.local_settings"`
3. Run 'python main.py' to start the web console back-end

## Testing
From `cloud-ingest/webconsole/back-end` run `python -m unittest discover`.

## App Engine Deployment
1. Make sure you have installed the [Google Cloud SDK](https://cloud.google.com/sdk/docs/)
2. Run `gcloud app deploy --project <your-cloud-project-id> app.<test|prod>.yaml`
   with the id of your Google Cloud project.
