CREATE TABLE JobConfigs (
     JobConfigId STRING(MAX) NOT NULL,
     JobSpec STRING(MAX),
   ) PRIMARY KEY(JobConfigId)

CREATE TABLE JobRuns (
     JobConfigId STRING(MAX) NOT NULL,
     JobRunId STRING(MAX) NOT NULL,
     Status INT64 NOT NULL,
     JobCreationTime INT64 NOT NULL,
     JobFinishTime INT64,
     Counters STRING(MAX) NOT NULL,
   ) PRIMARY KEY(JobConfigId, JobRunId),
   INTERLEAVE IN PARENT JobConfigs ON DELETE NO ACTION

CREATE TABLE Tasks (
      JobConfigId STRING(MAX) NOT NULL,
      JobRunId STRING(MAX) NOT NULL,
      TaskId STRING(MAX) NOT NULL,
      TaskSpec STRING(MAX) NOT NULL,
      TaskType INT64 NOT NULL,
      Status INT64 NOT NULL,
      CreationTime INT64 NOT NULL,
      WorkerId STRING(MAX),
      LastModificationTime INT64 NOT NULL,
      FailureType INT64,
      FailureMessage STRING(MAX),
    ) PRIMARY KEY(JobConfigId, JobRunId, TaskId),
    INTERLEAVE IN PARENT JobRuns ON DELETE NO ACTION

CREATE INDEX TasksByStatus ON Tasks(Status, JobConfigId, JobRunId)

CREATE NULL_FILTERED INDEX TasksByJobConfigIdJobRunIdFailureType
      ON Tasks(JobConfigId, JobRunId, FailureType), INTERLEAVE IN JobRuns

CREATE TABLE LogEntries (
      JobConfigId STRING(MAX) NOT NULL,
      JobRunId STRING(MAX) NOT NULL,
      TaskId STRING(MAX) NOT NULL,
      LogEntryId INT64 NOT NULL,
      CreationTime INT64 NOT NULL,
      CurrentStatus INT64 NOT NULL,
      PreviousStatus INT64 NOT NULL,
      FailureMessage STRING(MAX),
      LogEntry STRING(MAX) NOT NULL,
      Processed BOOL NOT NULL,
    ) PRIMARY KEY(JobConfigId, JobRunId, TaskId, LogEntryId),
    INTERLEAVE IN PARENT Tasks ON DELETE NO ACTION

CREATE INDEX LogEntriesByProcessed ON LogEntries(Processed)
