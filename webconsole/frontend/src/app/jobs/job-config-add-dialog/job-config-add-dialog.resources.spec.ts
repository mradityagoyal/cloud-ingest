import { TestBed, async } from '@angular/core/testing';
import { JobConfig } from '../jobs.resources';
import { JobConfigFormModel } from './job-config-add-dialog.resources';

let fakeJobConfigModel: JobConfigFormModel;

const FAKE_API_JOB_CONFIG1: JobConfig = {
  JobConfigId : 'fakeJobConfigId',
  JobSpec: '{"onPremSrcDirectory": "fake/file/system/dir",' +
            '"gcsBucket": "fakeGcsBucket",' +
            '"bigqueryDataset": "fakeBigqueryDataset",' +
            '"bigqueryTable": "fakeBigqueryTable"}',
};

const FAKE_API_JOB_CONFIG2: JobConfig = {
  JobConfigId : 'fakeJobConfigId',
  JobSpec: '{"onPremSrcDirectory": "fake/file/system/dir",' +
            '"gcsBucket": "fakeGcsBucket"}',
};

describe('JobConfigFormModel', () => {

  beforeEach(() => {
    fakeJobConfigModel = new JobConfigFormModel(
                              /**jobConfigId**/ 'fakeJobConfigId',
                             /**gcsBucket**/ 'fakeGcsBucket',
                             /**fileSystemDirectory**/
                                 'fake/file/system/dir',
                             /**bigqueryDataset**/ 'fakeBigqueryDataset',
                             /**bigqueryTable**/ 'fakeBigqueryTable');
  });

  it('toApiJobConfig should return a valid api configuration', () => {
    const jobConfig = fakeJobConfigModel.toApiJobConfig();
    expect(jobConfig.JobConfigId).toEqual(FAKE_API_JOB_CONFIG1.JobConfigId);
    expect(JSON.parse(jobConfig.JobSpec)).toEqual(JSON.parse(FAKE_API_JOB_CONFIG1.JobSpec));
  });

  it('toApiJobConfig should not add bigQuery info if dataset is null or undefined', () => {
    fakeJobConfigModel.bigqueryDataset = null;
    let jobConfig = fakeJobConfigModel.toApiJobConfig();
    expect(JSON.parse(jobConfig.JobSpec)).toEqual(JSON.parse(FAKE_API_JOB_CONFIG2.JobSpec));

    fakeJobConfigModel.bigqueryDataset = undefined;
    jobConfig = fakeJobConfigModel.toApiJobConfig();
    expect(JSON.parse(jobConfig.JobSpec)).toEqual(JSON.parse(FAKE_API_JOB_CONFIG2.JobSpec));
  });

  it('toApiJobConfig should not add bigQuery info if dataset is an empty string', () => {
    fakeJobConfigModel.bigqueryDataset = '';
    const jobConfig = fakeJobConfigModel.toApiJobConfig();
    expect(JSON.parse(jobConfig.JobSpec)).toEqual(JSON.parse(FAKE_API_JOB_CONFIG2.JobSpec));
  });

  it('toApiJobConfig should not add bigQuery info if table is null or undefined', () => {
    fakeJobConfigModel.bigqueryTable = null;
    let jobConfig = fakeJobConfigModel.toApiJobConfig();
    expect(JSON.parse(jobConfig.JobSpec)).toEqual(JSON.parse(FAKE_API_JOB_CONFIG2.JobSpec));

    fakeJobConfigModel.bigqueryTable = undefined;
    jobConfig = fakeJobConfigModel.toApiJobConfig();
    expect(JSON.parse(jobConfig.JobSpec)).toEqual(JSON.parse(FAKE_API_JOB_CONFIG2.JobSpec));
  });

  it('toApiJobConfig should not add bigQuery info if table is an empty string', () => {
    fakeJobConfigModel.bigqueryTable = '';
    const jobConfig = fakeJobConfigModel.toApiJobConfig();
    expect(JSON.parse(jobConfig.JobSpec)).toEqual(JSON.parse(FAKE_API_JOB_CONFIG2.JobSpec));
  });

  it('toApiJobConfig should throw error if job config id is null or undefined', () => {
    fakeJobConfigModel.jobConfigId = null;
    expect(fakeJobConfigModel.toApiJobConfig).toThrow();

    fakeJobConfigModel.jobConfigId = undefined;
    expect(fakeJobConfigModel.toApiJobConfig).toThrow();
  });

  it('toApiJobConfig should throw error if job config id is an empty string', () => {
    fakeJobConfigModel.jobConfigId = '';
    expect(fakeJobConfigModel.toApiJobConfig).toThrow();
  });

  it('toApiJobConfig should throw error if gcsBucket id is null or undefined', () => {
    fakeJobConfigModel.gcsBucket = null;
    expect(fakeJobConfigModel.toApiJobConfig).toThrow();

    fakeJobConfigModel.gcsBucket = undefined;
    expect(fakeJobConfigModel.toApiJobConfig).toThrow();
  });

  it('toApiJobConfig should throw error if gcsBucket id is an empty string', () => {
    fakeJobConfigModel.gcsBucket = '';
    expect(fakeJobConfigModel.toApiJobConfig).toThrow();
  });

  it('toApiJobConfig should throw error if fileSystemDirectory id is null or undefined', () => {
    fakeJobConfigModel.fileSystemDirectory = null;
    expect(fakeJobConfigModel.toApiJobConfig).toThrow();

    fakeJobConfigModel.fileSystemDirectory = undefined;
    expect(fakeJobConfigModel.toApiJobConfig).toThrow();
  });

  it('toApiJobConfig should throw error if fileSystemDirectory id is an empty string', () => {
    fakeJobConfigModel.fileSystemDirectory = '';
    expect(fakeJobConfigModel.toApiJobConfig).toThrow();
  });

});
