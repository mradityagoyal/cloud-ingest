var chai = require('chai')
var expect = chai.expect
var should = chai.should()
var assert = require('assert')
var sinon = require('sinon')
var gcs_to_bq = require('./../gcs_to_bq_importer_main')
var gcs_bq_helper= require('./../bq_helper_methods')

describe('CallBQImporter', function() {
  beforeEach(function() {
    sinon.stub(gcs_to_bq, 'CallBqImporter').resolves("");
  });

  afterEach(function() {
    gcs_to_bq.CallBqImporter.restore()
  });

  it('null pubsub event returns error', function(done) {
    gcs_to_bq.GcsToBq(null, function(err, res) {
      should.not.exist(res);
      should.exist(err);
    })
    done()
  });

  it('Missing task_id returns error', function(done) {
    var pubsub_data = {
      'src_gcs_bucket':'my_gcs_bucket',
      'src_gcs_object':'my_gcs_object',
      'dst_bq_dataset':'my_bq_dataset',
      'dst_bq_table':'my_bq_table'
    };
    var data_as_str = JSON.stringify(pubsub_data);
    var data_as_b64 = new Buffer(data_as_str).toString("base64");
    var event = {
      'data':{'data':data_as_b64}
    }

    gcs_to_bq.GcsToBq(event, function(err, res) {
      should.not.exist(res);
      should.exist(err);
      expect(err.message).to.equal("Pubsub payload has missing task_id");
    })
    done()
  });

  it('Simplest valid case', function(done) {
    var pubsub_data = {
      'task_id': 123,
      'src_gcs_bucket':'my_gcs_bucket',
      'src_gcs_object':'my_gcs_object',
      'dst_bq_dataset':'my_bq_dataset',
      'dst_bq_table':'my_bq_table'
    };
    var data_as_str = JSON.stringify(pubsub_data);
    var data_as_b64 = new Buffer(data_as_str).toString("base64");
    var event = {
      'data':{'data':data_as_b64}
    }

    gcs_to_bq.GcsToBq(event, function(err, res) {
      console.log(err)
      should.not.exist(err);
      should.exist(res);
      // args[4] === task_id
      expect(gcs_to_bq.CallBqImporter.getCall(0).args[4]).to.equal(123)
      // args[5] === bucket_name
      expect(gcs_to_bq.CallBqImporter.getCall(0).args[5]).to.equal('my_gcs_bucket')
      // args[6] === file_name
      expect(gcs_to_bq.CallBqImporter.getCall(0).args[6]).to.equal('my_gcs_object')
      // args[7] === dataset_id
      expect(gcs_to_bq.CallBqImporter.getCall(0).args[7]).to.equal('my_bq_dataset')
      // args[8] === table_id
      expect(gcs_to_bq.CallBqImporter.getCall(0).args[8]).to.equal('my_bq_table')
    })

    done()
  });
});

describe('PublishingSuccess', function() {
  beforeEach(function() {
    sinon.stub(gcs_bq_helper, 'importFileFromGCS').resolves("");
    sinon.stub(gcs_to_bq, 'PublishMessage').resolves("");
  });

  afterEach(function() {
    gcs_bq_helper.importFileFromGCS.restore()
    gcs_to_bq.PublishMessage.restore()
  });

  it('Publish success to PubSub', function(done) {
    var pubsub_data = {
      'task_id': 123,
      'src_gcs_bucket':'my_gcs_bucket',
      'src_gcs_object':'my_gcs_object',
      'dst_bq_dataset':'my_bq_dataset',
      'dst_bq_table':'my_bq_table'
    };
    var data_as_str = JSON.stringify(pubsub_data);
    var data_as_b64 = new Buffer(data_as_str).toString("base64");
    var event = {
      'data':{'data':data_as_b64}
    }

    gcs_to_bq.GcsToBq(event, function(err, res) {
      console.log(err)
      should.not.exist(err);
      should.exist(res);
      var message = gcs_to_bq.PublishMessage.getCall(0).args[2]
      expect(message.task_id).to.equal(123)
      expect(message.status).to.equal('SUCCESS')
    })

    done()
  });
});

describe('PublishingFailure', function() {
  beforeEach(function() {
    sinon.stub(gcs_bq_helper, 'importFileFromGCS').rejects(new Error("bad robot"));
    sinon.stub(gcs_to_bq, 'PublishMessage').resolves("");
  });

  afterEach(function() {
    gcs_bq_helper.importFileFromGCS.restore()
    gcs_to_bq.PublishMessage.restore()
  });

  it('Publish failure to PubSub', function(done) {
    var pubsub_data = {
      'task_id': 123,
      'src_gcs_bucket':'my_gcs_bucket',
      'src_gcs_object':'my_gcs_object',
      'dst_bq_dataset':'my_bq_dataset',
      'dst_bq_table':'my_bq_table'
    };
    var data_as_str = JSON.stringify(pubsub_data);
    var data_as_b64 = new Buffer(data_as_str).toString("base64");
    var event = {
      'data':{'data':data_as_b64}
    }

    gcs_to_bq.GcsToBq(event, function(err, res) {
      console.log(err)
      should.not.exist(err);
      should.exist(res);
      var message = gcs_to_bq.PublishMessage.getCall(0).args[2]
      expect(message.task_id).to.equal(123)
      expect(message.status).to.equal('FAILURE')
      expect(message.failure_message).to.equal('bad robot')
    })

    done()
  });
});

