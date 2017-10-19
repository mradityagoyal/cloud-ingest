import { JobConfig } from '../jobs.resources';

/**
 * This class contains the job configuration fields as intended for use in the form to add a job configuration.
 */
 export class JobConfigFormModel {
    constructor(
      public jobConfigId: string,
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
          this.gcsBucket == null || this.gcsBucket === '' ||
          this.fileSystemDirectory == null || this.fileSystemDirectory === '') {
          throw new Error('There are unset required fields.');
      }
      this.trimWhiteSpaces();
      let jobSpec: string =
                `{"onPremSrcDirectory": "${this.fileSystemDirectory}",` +
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

    private trimWhiteSpaces() {
      for (const property of ['jobConfigId', 'gcsBucket', 'fileSystemDirectory',
        'bigqueryDataset', 'bigqueryTable']) {
          if (this[property] != null) {
            this[property] = this[property].trim();
          }
      }
    }
 }
