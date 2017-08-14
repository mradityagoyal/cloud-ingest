let chai = require('chai');
let expect = chai.expect;
let should = chai.should();
let assert = chai.assert;
let sinon = require('sinon');
let bq_helper_methods = require('./../bq_helper_methods');
let BigQuery = require('@google-cloud/bigquery');
let Storage = require('@google-cloud/storage');

let FakeBigQuery;
let FakeStorage;
let storageStub;
let bigQueryStub;
let bigQueryTableStub;
let fakeSuccessJobResource;
let fakeBigqueryJobResponse;
let fakePendingJobResource;
let fakeAlreadyExistsError;

const FAKE_DATASET_ID = 'fake-datasetid';
const FAKE_TABLE_ID = 'fake-tableid';
const FAKE_BUCKET_NAME = 'fake-bucketname';
const FAKE_FILENAME = 'fake-filename';
const FAKE_PROJECTID = 'fake-projectid';
const FAKE_TASKID = 'fake-taskid';
const FAKE_TASKID2 = 'fake-taskid2';
const FAKE_TASKID3 = 'fake-taskid3-dfd8392Df39ui20D';
// The md5 hash of the string 'fake-taskid'
const FAKE_TASKID_MD5_HASH = '09a27255587039dad6854866769657c8';
// The md5 hash of the string 'fake-taskid2'
const FAKE_TASKID2_MD5_HASH = 'dd41fe736b09ab014095569db60d9638';

describe('importFileFromGCS', function() {
  beforeEach(function() {
    FakeBigQuery = sinon.stub();
    FakeStorage = sinon.stub();

    storageStub = sinon.createStubInstance(Storage);
    storageBucketStub = sinon.createStubInstance(Storage.Bucket);
    storageFileStub = sinon.createStubInstance(Storage.File);
    bigQueryStub = sinon.createStubInstance(BigQuery);
    bigQueryTableStub = sinon.createStubInstance(BigQuery.Table);
    bigQueryDatasetStub = sinon.createStubInstance(BigQuery.Dataset);

    FakeStorage.withArgs({projectId: FAKE_PROJECTID}).returns(storageStub);
    FakeBigQuery.withArgs({projectId: FAKE_PROJECTID}).returns(bigQueryStub);

    bigQueryStub.dataset.withArgs(FAKE_DATASET_ID).returns(bigQueryDatasetStub);
    bigQueryDatasetStub.table.withArgs(FAKE_TABLE_ID)
        .returns(bigQueryTableStub);
    storageStub.bucket.withArgs(FAKE_BUCKET_NAME).returns(storageBucketStub);
    storageBucketStub.file.withArgs(FAKE_FILENAME).returns(storageFileStub);
    bigQueryTableStub.bigQuery = bigQueryStub;
    bigQueryTableStub.dataset = bigQueryDatasetStub;

    fakeSuccessJobResource = {
      id: 'job_project_bucket:' + FAKE_TASKID_MD5_HASH,
      jobReference: {jobId: FAKE_TASKID_MD5_HASH},
      status: {errors: [], state: 'DONE'}
    };
    fakeErrorJobResource = {
      id: 'job_project_bucket:' + FAKE_TASKID_MD5_HASH,
      jobReference: {jobId: FAKE_TASKID_MD5_HASH},
      status: {errors: ['FakeErrorMessage'], state: 'DONE'}
    };
    fakePendingJobResource = {
      id: 'job_project_bucket:' + FAKE_TASKID_MD5_HASH,
      jobReference: {jobId: FAKE_TASKID_MD5_HASH},
      status: {errors: [], state: 'PENDING'}
    };
    fakeBigqueryJobResponse = {};

    fakeAlreadyExistsError = {code: 409, errorMessage: 'job already exists'};

    storageFileStub.bucket = {name: FAKE_BUCKET_NAME};
    storageFileStub.name = FAKE_FILENAME;
    bigQueryTableStub.projectId = FAKE_PROJECTID;
    bigQueryTableStub.dataset = {id: FAKE_DATASET_ID};
    bigQueryStub.projectId = FAKE_PROJECTID;
    bigQueryDatasetStub.id = FAKE_DATASET_ID;
    bigQueryTableStub.id = FAKE_TABLE_ID;

    // Instant polling rate for unit test speed.
    bq_helper_methods.CHECK_JOB_DONE_POLLING_RATE_MILLISECONDS = 0;
  });


  it('should make one call to bigquery', function(done) {
    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID);
    assert(bigQueryStub.request.calledOnce);
    done();
  });

  it('should make a POST request to the /jobs uri in bigquery with a job resource',
     function(done) {
       bq_helper_methods.importFileFromGCS(
           FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
           FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID);
       let bigQueryArgs = bigQueryStub.request.getCall(0).args[0];
       expect(bigQueryArgs.method).to.equal('POST');
       expect(bigQueryArgs.uri).to.equal('/jobs');
       assert.isDefined(bigQueryArgs.json);
       done();
     });

  it('should set the jobId as the md5 hash of the taskid', function(done) {
    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID);
    let bigQueryArgs = bigQueryStub.request.getCall(0).args[0];
    let jobResourceJson = bigQueryArgs.json;
    expect(jobResourceJson.jobReference.jobId)
        .to.equal(FAKE_TASKID_MD5_HASH);
    expect(jobResourceJson.jobReference.projectId).to.equal(FAKE_PROJECTID);


    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID2);
    bigQueryArgs = bigQueryStub.request.getCall(1).args[0];
    jobResourceJson = bigQueryArgs.json;
    expect(jobResourceJson.jobReference.jobId)
        .to.equal(FAKE_TASKID2_MD5_HASH);
    expect(jobResourceJson.jobReference.projectId).to.equal(FAKE_PROJECTID);

    done();
  });

  it('should set the writeDisposition as WRITE_APPEND', function(done) {
    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID);
    let bigQueryArgs = bigQueryStub.request.getCall(0).args[0];
    let jobResourceJson = bigQueryArgs.json;
    expect(jobResourceJson.configuration.load.writeDisposition)
        .to.equal('WRITE_APPEND');
    done();
  });

  it('should set the sourceUri for the import file', function(done) {
    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID);
    let bigQueryArgs = bigQueryStub.request.getCall(0).args[0];
    let jobResourceJson = bigQueryArgs.json;
    expect(jobResourceJson.configuration.load.sourceUris)
        .to.contain('gs://' + FAKE_BUCKET_NAME + '/' + FAKE_FILENAME);
    done();
  });

  it('should set the destination table for the bigquery table', function(done) {
    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID);
    let bigQueryArgs = bigQueryStub.request.getCall(0).args[0];
    let jobResourceJson = bigQueryArgs.json;
    expect(jobResourceJson.configuration.load.destinationTable.projectId)
        .to.equal(FAKE_PROJECTID);
    expect(jobResourceJson.configuration.load.destinationTable.datasetId)
        .to.equal(FAKE_DATASET_ID);
    expect(jobResourceJson.configuration.load.destinationTable.tableId)
        .to.equal(FAKE_TABLE_ID);
    done();
  });

  it('should resolve with job if the bigquery job request succeeds',
     function(done) {

       // Bigquery responds to request call with null error and fake response
       bigQueryStub.request.callsArgWith(1, null, fakeSuccessJobResource);
       bigQueryStub.job.withArgs(FAKE_TASKID_MD5_HASH)
           .returns(fakeBigqueryJobResponse);

       bq_helper_methods
           .importFileFromGCS(
               FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
               FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID)
           .then((response) => {
             expect(response).to.equal(fakeSuccessJobResource);
             done();
           })
           .catch((error) => {
             assert.fail(
                 job, error, 'The promise rejected when bigquery succeeded.');
           });
     });

  it('should reject with job if the bigquery job request fails',
     function(done) {
       bigQueryStub.request.callsArgWith(
           1, {errorMessage: 'fakeErrorMessage'}, fakeSuccessJobResource);

       bq_helper_methods
           .importFileFromGCS(
               FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
               FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID)
           .then((job) => {
             assert.fail(
                 job, null, 'The promise resolved when bigquery had error.');
           })
           .catch((error) => {
             done();
           });
     });

  it('should request with a valid jobResourceJson', function(done) {
    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID);
    let bigQueryArgs = bigQueryStub.request.getCall(0).args[0];
    let jobResourceJson = bigQueryArgs.json;
    expect(jobResourceJson.jobReference.jobId)
        .to.equal(FAKE_TASKID_MD5_HASH);
    expect(jobResourceJson.jobReference.projectId).to.equal(FAKE_PROJECTID);
    expect(jobResourceJson.configuration.load.writeDisposition)
        .to.equal('WRITE_APPEND');
    expect(jobResourceJson.configuration.load.sourceUris)
        .to.contain('gs://' + FAKE_BUCKET_NAME + '/' + FAKE_FILENAME);
    expect(jobResourceJson.configuration.load.destinationTable.projectId)
        .to.equal(FAKE_PROJECTID);
    expect(jobResourceJson.configuration.load.destinationTable.datasetId)
        .to.equal(FAKE_DATASET_ID);
    expect(jobResourceJson.configuration.load.destinationTable.tableId)
        .to.equal(FAKE_TABLE_ID);
    done();
  });

  /**
   * Checks that the jobId created is the same for a constant input task id.
   */
  it('should have the same job id for the same task id', function(done) {
    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID3);
    let bigQueryArgs = bigQueryStub.request.getCall(0).args[0];
    let jobResourceJson = bigQueryArgs.json;
    let jobId1 = jobResourceJson.jobReference.jobId;

    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID3);
    bigQueryArgs = bigQueryStub.request.getCall(1).args[0];
    jobResourceJson = bigQueryArgs.json;
    let jobId2 = jobResourceJson.jobReference.jobId;

    expect(jobId1).to.equal(jobId2);
    done();
  });

  it('should create a new job with a valid job id', function(done) {
    bq_helper_methods.importFileFromGCS(
        FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
        FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID);
    let bigQueryArgs = bigQueryStub.request.getCall(0).args[0];
    let jobResourceJson = bigQueryArgs.json;
    expect(jobResourceJson.jobReference.jobId)
        .to.match(
            /([A-Z]|[a-z]|\d|(-)|(_))+/,
            'The job id is invalid. It must only contain letters, numbers,' +
                'underscores or dashes. See ' +
                'https://cloud.google.com/bigquery/docs/reference/rest/v2/jobs#configuration.load');
    expect(jobResourceJson.jobReference.jobId).to.have.lengthOf.below(1025);
    done();
  });

  it('should resolve with the job metadata if the job is done and succeeded',
     function(done) {
       bigQueryStub.request.callsArgWith(
           1, fakeAlreadyExistsError, fakeSuccessJobResource);
       fakeBigqueryJobResponse.getMetadata =
           sinon.stub().resolves([fakeSuccessJobResource]);
       bigQueryStub.job.withArgs(FAKE_TASKID_MD5_HASH)
           .returns(fakeBigqueryJobResponse);

       bq_helper_methods
           .importFileFromGCS(
               FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
               FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID)
           .then((response) => {
             expect(response).to.equal(fakeSuccessJobResource);
             done();
           })
           .catch((error) => {
             assert.fail(
                 job, error,
                 'The promised rejected when the job already existed');
           });
     });

  it('should wait for the job if the job is pending and succeeded',
     function(done) {
       bigQueryStub.request.callsArgWith(
           1, fakeAlreadyExistsError, fakeSuccessJobResource);
       fakeBigqueryJobResponse.getMetadata =
           sinon.stub()
               .onFirstCall()
               .resolves([fakePendingJobResource])
               .onSecondCall()
               .resolves([fakeSuccessJobResource]);
       bigQueryStub.job.withArgs(FAKE_TASKID_MD5_HASH)
           .returns(fakeBigqueryJobResponse);

       bq_helper_methods
           .importFileFromGCS(
               FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
               FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID)
           .then((response) => {
             expect(response).to.equal(fakeSuccessJobResource);
             done();
           })
           .catch((error) => {
             assert.fail(
                 job, error,
                 'The promised rejected when the job already existed');
           });
     });

  it('should reject with the job metadata if the job done and failed',
     function(done) {
       bigQueryStub.request.callsArgWith(
           1, fakeAlreadyExistsError, fakeErrorJobResource);
       fakeBigqueryJobResponse.getMetadata =
           sinon.stub().resolves([fakeErrorJobResource]);
       bigQueryStub.job.withArgs(FAKE_TASKID_MD5_HASH)
           .returns(fakeBigqueryJobResponse);

       bq_helper_methods
           .importFileFromGCS(
               FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
               FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID)
           .then((response) => {
             assert.fail(
                 response, null,
                 'The promise resolved when there was an error in existing job');
           })
           .catch((error) => {
             expect(error).to.equal(fakeErrorJobResource);
             done();
           });
     });

  it('should reject with job metadata if the job is pending and failed',
     function(done) {
       bigQueryStub.request.callsArgWith(
           1, fakeAlreadyExistsError, fakeErrorJobResource);
       fakeBigqueryJobResponse.getMetadata =
           sinon.stub()
               .onFirstCall()
               .resolves([fakePendingJobResource])
               .onSecondCall()
               .resolves([fakeErrorJobResource]);
       bigQueryStub.job.withArgs(FAKE_TASKID_MD5_HASH)
           .returns(fakeBigqueryJobResponse);

       bq_helper_methods
           .importFileFromGCS(
               FakeBigQuery, FakeStorage, FAKE_DATASET_ID, FAKE_TABLE_ID,
               FAKE_BUCKET_NAME, FAKE_FILENAME, FAKE_PROJECTID, FAKE_TASKID)
           .then((response) => {
             assert.fail(
                 response, null,
                 'The promise resolved with there was an error in the pending job');
           })
           .catch((error) => {
             expect(error).to.equal(fakeErrorJobResource);
             done();
           });
     });
});
