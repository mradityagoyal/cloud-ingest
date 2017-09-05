import { JobConfig } from './api.resources';

/**
 * This class contains the job configuration fields as intended for use in the form to add a job configuration.
 */
 export class JobConfigFormModel {
    constructor(
      public jobConfigId: string,
      public gcsDirectory: string,
      public gcsBucket: string,
      // The directory on the on-premises file system.
      public fileSystemDirectory: string,
      public bigqueryDataset: string,
      public bigqueryTable: string
    ) {}

   /**
    * Converts the Job Configuration in the class to the job configuration in the API.
    */
   toApiJobConfig(): JobConfig {
      // If the required fields are not set, throw an error.
      if (this.jobConfigId == null || this.jobConfigId === '' ||
          this.gcsDirectory == null || this.gcsDirectory === '' ||
          this.gcsBucket == null || this.gcsBucket === '' ||
          this.fileSystemDirectory == null || this.fileSystemDirectory === '') {
          throw new Error('There are unset required fields.');
      }
      let jobSpec: string = `{"gcsDirectory": "${this.gcsDirectory}", ` +
                 `"onPremSrcDirectory": "${this.fileSystemDirectory}",` +
                 `"gcsBucket": "${this.gcsBucket}"`;
      // If BigQuery information is present, add it to the JobSpec.
      if (this.bigqueryTable != null && this.bigqueryTable !== '' &&
          this.bigqueryDataset != null && this.bigqueryDataset !== '') {

          jobSpec = jobSpec + `, "bigqueryTable": "${this.bigqueryTable}",` +
                    `"bigqueryDataset": "${this.bigqueryDataset}"`;
      }
      jobSpec = jobSpec + '}';
      return {
        JobConfigId: this.jobConfigId,
        JobSpec: jobSpec,
      };
    }
 }
