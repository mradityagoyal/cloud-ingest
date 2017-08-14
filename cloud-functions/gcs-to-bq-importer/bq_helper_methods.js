'use strict';

let crypto = require('crypto');
let extend = require('extend');

/**
 * Error code indicating that the big query job already exists. Taken from:
 * https://cloud.google.com/bigquery/troubleshooting-errors
 */
const BQ_ALREADY_EXISTS = 409;

/**
 * Time in milliseconds before the the cloud function checks if an existing job
 * has finished. This variable is exported so that it can be read and modified
 * by the unit tests.
 */
exports.CHECK_JOB_DONE_POLLING_RATE_MILLISECONDS = 15000;

/**
 * Imports a file from GCS into BigQuery.
 *
 * @param {Object} BigQuery The BigQuery library to make an instance of
 *     bigquery and interact with the bigquery api.
 * @param {Object} Storage The Google Storage library to make an instance of
 *     storage and interact with the storage api.
 * @param {string} datasetId The dataset to import the file into.
 * @param {string} tableId The table to import the file into.
 * @param {string} bucketName The name of the bucket where the file is.
 * @param {string} fileName The name of the file to import.
 * @param {string} projectId The project where the import will happen.
 * @param {string} taskId Used to assign a unique import job id.
 */
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
      table, storage.bucket(bucketName).file(fileName), bqResourceJobConfig)
      .then((results) => {
        job = results[0].metadata;
        handleJobResult(job, resolve, reject);
      })
      .catch((err) => {
        if (err.code == BQ_ALREADY_EXISTS) {
          waitUntilJobCompletes(bigquery, importJobId)
              .then((results) => {
                handleJobResult(results, resolve, reject);
              })
              .catch((error) => {
                reject(error);
              });
        } else {
          console.error(`BQ-import had ERROR:`, err);
          reject(err);
        }
      });
});

/**
 * Waits until a job completes and gets the job metadata. Returns a promise. If
 * the promise resolves, it contains a resource job object with the job
 * metadata. If the promise rejects, it contains the error while obtaining the
 * job metadata.
 *
 * @param {BigQuery} bigquery The bigquery instance to query the job.
 * @param {string} jobId The job id to query the job.
 * @returns {Promise} A promise with the job metadata or error.
 */
const waitUntilJobCompletes = function(bigquery, jobId) {
  return new Promise((resolve, reject) => {
    const checkHasJobFinished = function() {
      let job = bigquery.job(jobId);
      job.getMetadata()
          .then((response) => {
            let jobMetadata = response[0];
            if (jobMetadata.status.state == 'DONE') {
              resolve(jobMetadata);
            } else {
              // If the job still is not done, check again later.
              setTimeout(
                  checkHasJobFinished,
                  exports.CHECK_JOB_DONE_POLLING_RATE_MILLISECONDS);
            }
          })
          .catch((error) => {
            reject(error);
          });
    };
    checkHasJobFinished();
  });
};

/**
 * Handles the result of a job. If the job has an error, it will call the
 * input reject parameter. Otherwise, it will call the input resolve parameter.
 *
 * @param {Object} job The bigquery job resource.
 * @param {Function} resolve The promise resolve function.
 * @param {Function} reject The promise reject function.
 */
const handleJobResult = function(job, resolve, reject) {
  if (job.status.errors && job.status.errors.length > 0) {
    console.error(`BQ-import job ${job.id} had ERROR:`,
        job.status.errors[0]);
    reject(job);
  } else {
    resolve(job);
  }
};

/**
 * Creates a single file import job on the input table from the input sourceFile
 * with the input job resource representation.
 *
 * @param {BigQuery.Table} table The BigQuery table where the function creates
 *     the import job.
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
      } else {
        let job = table.bigQuery.job(resp.jobReference.jobId);
        job.metadata = resp;
        resolve([job, resp]);
      }
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
