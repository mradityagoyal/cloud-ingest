'use strict';

// A function to import an object from GCS into BigQuery.
// TODO: This method is brittle and not idempotent. If a pub-sub message
// gets redelivered, it will result in another BQ job. One solution is
// for this method to take in a unique task_id and use that in creation
// of a BQ-job. Then check if that job is already running and if so do
// nothing (return).
exports.importFileFromGCS =
  (datasetId, tableId, bucketName, fileName, projectId) =>
    new Promise((resolve, reject) => {
  const BigQuery = require('@google-cloud/bigquery');
  const Storage = require('@google-cloud/storage');

  // Instantiates clients
  const bigquery = BigQuery({
    projectId: projectId
  });

  const storage = Storage({
    projectId: projectId
  });

  let job;
  // The configurations should follow configuration.load
  // https://cloud.google.com/bigquery/docs/reference/rest/v2/jobs#configuration.load
  const bq_load_config = {
    "writeDisposition": "WRITE_APPEND"
  }

  // Imports data from a Google Cloud Storage file into the table
  bigquery
    .dataset(datasetId)
    .table(tableId)
    .import(storage.bucket(bucketName).file(fileName), bq_load_config)
    .then((results) => {
      job = results[0];
      console.log(`BQ-import job ${job.id} started.`);
      return job.promise();
    })
    .then((results) => {  // Job has finished at this point.
      job = results[0];
      // Check status of completed job.
      if (job.status.errors && job.status.errors.length > 0) {
        console.error(`BQ-import job ${job.id} had ERROR:`, job.status.errors[0]);
        reject(job);
      } else {
        console.log(`BQ-import job ${job.id} completed`);
        resolve(job);
      }
    })
    .catch((err) => {
      console.error(`BQ-import had ERROR:`, err);
      reject(job)
    });
});
