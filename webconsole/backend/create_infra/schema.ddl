CREATE TABLE Projects (
    ProjectId STRING(MAX) NOT NULL,
    ProjectInfo STRING(MAX),
) PRIMARY KEY(ProjectId)

CREATE TABLE JobConfigs (
    ProjectId STRING(MAX) NOT NULL,
    JobConfigId STRING(MAX) NOT NULL,
    JobSpec STRING(MAX),
) PRIMARY KEY(ProjectId, JobConfigId)

CREATE TABLE JobRuns (
    ProjectId STRING(MAX) NOT NULL,
    JobConfigId STRING(MAX) NOT NULL,
    JobRunId STRING(MAX) NOT NULL,
    Status INT64 NOT NULL,
    JobCreationTime INT64 NOT NULL,
    JobFinishTime INT64,
    Counters STRING(MAX) NOT NULL,
) PRIMARY KEY(ProjectId, JobConfigId, JobRunId),
INTERLEAVE IN PARENT JobConfigs ON DELETE CASCADE

CREATE TABLE Tasks (
    ProjectId STRING(MAX) NOT NULL,
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
) PRIMARY KEY(ProjectId, JobConfigId, JobRunId, TaskId),
INTERLEAVE IN PARENT JobRuns ON DELETE CASCADE

CREATE INDEX TasksByStatus ON Tasks(Status, ProjectId, JobConfigId, JobRunId)

CREATE NULL_FILTERED INDEX TasksByJobConfigIdJobRunIdFailureType
    ON Tasks(ProjectId, JobConfigId, JobRunId, FailureType), INTERLEAVE IN JobRuns

CREATE TABLE LogEntries (
    ProjectId STRING(MAX) NOT NULL,
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
) PRIMARY KEY(ProjectId, JobConfigId, JobRunId, TaskId, LogEntryId),
INTERLEAVE IN PARENT Tasks ON DELETE CASCADE

CREATE INDEX LogEntriesByProcessed ON LogEntries(Processed)
