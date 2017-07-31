# Ingest Web Console

## Environment Set-up
1. Install virtualenv, create an environment, and activate the newly created environment.
2. Run `pip install -t lib -r requirements.txt` to install the requirements for the backend Flask application.

## Local Deployment
1. Create a service account for your Ingest Cloud Spanner database
2. Generate a JSON key for the service account and store it locally
3. Create a settings file (such as ingestwebconsole.local_settings) and
   store the following values in it:
   ```
   JSON_KEY_PATH='<path to your json key>'
   SPANNER_INSTANCE='<the id of your Cloud Spanner instance>'
   SPANNER_DATABASE='<the name of the Spanner database>'
   ```

   Here is an example file:
   ```
   JSON_KEY_PATH='/usr/home/awesomeuser/Documents/webconsole/awesomeuser-project-a2cdf6d63b12.json'
   SPANNER_INSTANCE='spanner-instance'
   SPANNER_DATABASE='database'
   ```
4. Set the `INGEST_CONFIG_PATH` environment variable to hold the path to
   your local config file.
Linux: `export INGEST_CONFIG_PATH="example/path/to/ingestwebconsole.local_settings"`
5. Run 'python main.py' to start the web console back-end

## Testing
From `cloud-ingest/webconsole/back-end` run `python -m unittest discover`.
