'use strict';

let crypto = require('crypto');
let extend = require('extend');

// A function to import an object from GCS into BigQuery.
// TODO: This method is brittle and not idempotent. If a pub-sub message
// gets redelivered, it will result in another BQ job. One solution is
// for this method to take in a unique task_id and use that in creation
// of a BQ-job. Then check if that job is already running and if so do
// nothing (return).
exports.importFileFromGCS =
  (BigQuery, Storage, datasetId, tableId, bucketName, fileName, projectId,
      taskId) =>
    new Promise((resolve, reject) => {

  // Instantiates clients
  const bigquery = BigQuery({
    projectId: projectId
  });

  const storage = Storage({
    projectId: projectId
  });

  let job;
  let importJobId = crypto.createHash('md5').update(taskId).digest("hex");

  let bqResourceJobConfig = {
    "jobReference": {
      "projectId": projectId,
      "jobId": importJobId
    },
    "configuration": {
      "load" : {
        "writeDisposition": "WRITE_APPEND"
      }
    }
  };

  let table = bigquery.dataset(datasetId).table(tableId);
  createSingleFileImportJobWithJobResource(
    table,
    storage.bucket(bucketName).file(fileName),
    bqResourceJobConfig)
    .then((results) => {
      job = results[0];
      console.log(`BQ-import job ${job.id} started.`);
      return job.promise();
    })
    .then((results) => {  // Job has finished at this point.
      job = results[0];
      // Check status of completed job.
      if (job.status.errors && job.status.errors.length > 0) {
        console.error(`BQ-import job ${job.id} had ERROR:`,
            job.status.errors[0]);
        reject(job);
      } else {
        // TODO(b/63799174): If the failed job status is job exists, then wait
        // until the job finishes and resolve or reject the promise.
        console.log(`BQ-import job ${job.id} completed`);
        resolve(job);
      }
    })
    .catch((err) => {
      console.error(`BQ-import had ERROR:`, err);
      reject(job);
    });
});

/**
 * Creates a single file import job on the input table from the input sourceFile
 * with the input job resource representation.
 *
 * @param {BigQuery.Table} table The BigQuery table where the function creates the
 *     import job.
 * @param {Storage.File} sourceFile The source file to import from.
 * @param {Object} jobResource The resource representation of the job. Must
 *     match the object structure in
 *     https://cloud.google.com/bigquery/docs/reference/rest/v2/jobs
 * @return {Object} A promise of the import job. If the promise rejects, it
 *     contains an error object. If it resolves, it contains an array of two
 *     items: the job and the response.
 */
const createSingleFileImportJobWithJobResource =
    function(table, sourceFile, jobResource) {
      return new Promise((resolve, reject) => {
    jobResource = jobResource || {};
    let body = getJobResourceBodyFromTableAndSourceFile(table, sourceFile);
    extend(true, body, jobResource);

    table.bigQuery.request({
      method: 'POST',
      uri: '/jobs',
      json: body
    }, (err, resp) => {
      if (err) {
        reject(err);
      }
      let job = table.bigQuery.job(resp.jobReference.jobId);
      job.metadata = resp;
      resolve([job, resp]);
    });
  });
};

/**
 * Gets the basic job resource json for an import job from the input table and
 * the input sourceFile.
 *
 * @param {BigQuery.Table} table The function reads the projectId, datasetId and
 *     tableId from here.
 * @param {Storage.File} sourceFile The source file to import.
 * @return {Object} A basic job resource json with the fields needed for the
 *     import job.
 */
const getJobResourceBodyFromTableAndSourceFile = function(table, sourceFile) {
  return {
    configuration: {
      load: {
        sourceUris: ['gs://' + sourceFile.bucket.name + '/' + sourceFile.name],
        destinationTable: {
          projectId: table.bigQuery.projectId,
          datasetId: table.dataset.id,
          tableId: table.id
        }
      }
    }
  };
};
