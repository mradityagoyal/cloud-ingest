'use strict'

// This module will run as a Google Cloud Function that is trigerred from
// a PubSub queue. The payload in the PubSub message will contain details on
// (1) the GCS object whose contents will be imported into a
// (2) BigQuery table.

const BQImporter = require('./bq_helper_methods')
const PubSub = require('@google-cloud/pubsub');

// TODO: take as a constant during creation of cloud function.
const resultTopicName = 'loadbigquery_progress'
const projectId = process.env.GCLOUD_PROJECT;

// Publishes a 'message' to a 'pub_sub' 'topic'.
exports.PublishMessage = (pub_sub, topic, message) => new Promise((resolve, reject) => {
  pub_sub.topic(topic)
    .publish(message)
    .then((data) => {
      resolve(`publishing to ${topic} with ${message} was successful`);
    })
    .catch((err) => {
      reject(`publishing to ${topic} with ${message} failed`);
    });
});

// Internal helper method to publish a failure message on 'pub_sub' 'topic'
// with 'task_id'.
const PublishErrorMessage = (pub_sub, topic, task_id, err_message) =>
    new Promise((resolve, reject) => {
  var failure = {
    task_id : task_id,
    status: 'FAILURE',
    failure_message: err_message
  }
  exports.PublishMessage(pub_sub, topic, failure)
    .then((res) => resolve(res))
    .catch((err) => reject(err));
});

// Internal helper method to publish a success message on 'pub_sub' 'topic'
// with 'task_id'.
const PublishSuccessMessage = (pub_sub, topic, task_id) =>
    new Promise((resolve, reject) => {
  var failure = {
    task_id : task_id,
    status: 'SUCCESS'
  }
  exports.PublishMessage(pub_sub, topic, failure)
    .then((res) => resolve(res))
    .catch((err) => reject(err));
});

// Helper method that actually makes the import call and then publishes
// appropriate (success/failure) message onto 'pub_sub'.
exports.CallBqImporter = (bq_importer, pub_sub, topic, project_id, task_id,
                          bucket_name, file_name, dataset_id, table_id) =>
    new Promise((resolve, reject) => {
  console.log(`Processing task_id: ${task_id}`);
  bq_importer
    .importFileFromGCS(dataset_id, table_id, bucket_name, file_name, project_id)
    .then((results) => {
      // BQ-Import was successful, so publish a message to progress topic
      // that import is done.
      PublishSuccessMessage(pub_sub, topic, task_id)
        .then((res) => { resolve(res) })
        .catch((err) => { reject(err) })
    })
    .catch((err) => {
      console.error("BQ-import job failed; publishing failure message to " +
                    "pub-sub-q.");
      PublishErrorMessage(pub_sub, topic, task_id, err.message)
        .then((res) => { resolve(res) })
        .catch((err) => { reject(err) })
    })
});

const ExtractAndValidatePayload = (pubsub_event) =>
    new Promise((resolve, reject) => {
  let json_payload = '';
  const pubsub_message = pubsub_event.data;
  const task_payload = pubsub_message.data ?
                       Buffer.from(pubsub_message.data, 'base64').toString() : ''
  if (task_payload === '') {
    reject(new Error('Empty pubsub payload'));
  }
  try {
    json_payload = JSON.parse(task_payload);
  } catch (SyntaxError) {
    reject(new Error(`Error parsing pubsub message ${e.message} in file
                     ${e.fileName}`));
  }
  if (json_payload === '') {
    reject(new Error("Empty JSON payload inside pub_sub event."))
  }

  // Extract out relevant fields from pubsub payload and make sure they are
  // all valid.
  const task_id = json_payload['task_id']
  if (!task_id) {
    reject(new Error("Pubsub payload has missing task_id"))
  }


  // TODO: below four error conditions should be published on pub-sub queue.
  const bucket_name = json_payload['src_gcs_bucket']
  if (!bucket_name) {
    reject(new Error(`Pubsub message with task_id ${task_id} has no
                     src_gcs_bucket`))
  }

  const file_name = json_payload['src_gcs_object']
  if (!file_name) {
    reject(new Error(`Pubsub message with task_id ${task_id} has no
                        src_gcs_object`))
  }

  const dataset_id = json_payload['dst_bq_dataset']
  if (!dataset_id) {
    reject(new Error(`Pubsub message with task_id ${task_id} has no
                     dst_bq_dataset`))
  }

  const table_id = json_payload['dst_bq_table']
  if (!table_id) {
    reject(new Error(`Pubsub message with task_id ${task_id} has no
                     dst_bq_table`))
  }

  // Valid payload, return it.
  resolve(json_payload)
});

exports.GcsToBq = function(pubsub_event, callback) {
  ExtractAndValidatePayload(pubsub_event)
    .then((payload) => {
        exports.CallBqImporter(BQImporter, PubSub, resultTopicName, projectId,
                               payload.task_id, payload.src_gcs_bucket,
                               payload.src_gcs_object, payload.dst_bq_dataset,
                               payload.dst_bq_table)
          .then((p) => { callback(null, "success") })
          .catch((err) => { callback(err) })
     })
    .catch((err) => { callback(err) })
}
