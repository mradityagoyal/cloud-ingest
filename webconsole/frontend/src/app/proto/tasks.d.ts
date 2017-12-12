import * as $protobuf from "protobufjs";

/** Properties of a TaskFailureType. */
export interface ITaskFailureType {
}

/** Represents a TaskFailureType. */
export class TaskFailureType implements ITaskFailureType {

    /**
     * Constructs a new TaskFailureType.
     * @param [properties] Properties to set
     */
    constructor(properties?: ITaskFailureType);

    /**
     * Creates a new TaskFailureType instance using the specified properties.
     * @param [properties] Properties to set
     * @returns TaskFailureType instance
     */
    public static create(properties?: ITaskFailureType): TaskFailureType;

    /**
     * Encodes the specified TaskFailureType message. Does not implicitly {@link TaskFailureType.verify|verify} messages.
     * @param message TaskFailureType message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: ITaskFailureType, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified TaskFailureType message, length delimited. Does not implicitly {@link TaskFailureType.verify|verify} messages.
     * @param message TaskFailureType message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: ITaskFailureType, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a TaskFailureType message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns TaskFailureType
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): TaskFailureType;

    /**
     * Decodes a TaskFailureType message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns TaskFailureType
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): TaskFailureType;

    /**
     * Verifies a TaskFailureType message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a TaskFailureType message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns TaskFailureType
     */
    public static fromObject(object: { [k: string]: any }): TaskFailureType;

    /**
     * Creates a plain object from a TaskFailureType message. Also converts values to other types if specified.
     * @param message TaskFailureType
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: TaskFailureType, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this TaskFailureType to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };
}

export namespace TaskFailureType {

    /** Type enum. */
    enum Type {
        UNUSED = 0,
        UNKNOWN = 1,
        FILE_MODIFIED_FAILURE = 2,
        MD5_MISMATCH_FAILURE = 3,
        PRECONDITION_FAILURE = 4,
        FILE_NOT_FOUND_FAILURE = 5,
        PERMISSION_FAILURE = 6
    }
}

/** Properties of a TaskStatus. */
export interface ITaskStatus {
}

/** Represents a TaskStatus. */
export class TaskStatus implements ITaskStatus {

    /**
     * Constructs a new TaskStatus.
     * @param [properties] Properties to set
     */
    constructor(properties?: ITaskStatus);

    /**
     * Creates a new TaskStatus instance using the specified properties.
     * @param [properties] Properties to set
     * @returns TaskStatus instance
     */
    public static create(properties?: ITaskStatus): TaskStatus;

    /**
     * Encodes the specified TaskStatus message. Does not implicitly {@link TaskStatus.verify|verify} messages.
     * @param message TaskStatus message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: ITaskStatus, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified TaskStatus message, length delimited. Does not implicitly {@link TaskStatus.verify|verify} messages.
     * @param message TaskStatus message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: ITaskStatus, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a TaskStatus message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns TaskStatus
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): TaskStatus;

    /**
     * Decodes a TaskStatus message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns TaskStatus
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): TaskStatus;

    /**
     * Verifies a TaskStatus message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a TaskStatus message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns TaskStatus
     */
    public static fromObject(object: { [k: string]: any }): TaskStatus;

    /**
     * Creates a plain object from a TaskStatus message. Also converts values to other types if specified.
     * @param message TaskStatus
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: TaskStatus, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this TaskStatus to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };
}

export namespace TaskStatus {

    /** Type enum. */
    enum Type {
        UNQUEUED = 0,
        QUEUED = 1,
        FAILED = 2,
        SUCCESS = 3
    }
}

/** Properties of a TaskType. */
export interface ITaskType {
}

/** Represents a TaskType. */
export class TaskType implements ITaskType {

    /**
     * Constructs a new TaskType.
     * @param [properties] Properties to set
     */
    constructor(properties?: ITaskType);

    /**
     * Creates a new TaskType instance using the specified properties.
     * @param [properties] Properties to set
     * @returns TaskType instance
     */
    public static create(properties?: ITaskType): TaskType;

    /**
     * Encodes the specified TaskType message. Does not implicitly {@link TaskType.verify|verify} messages.
     * @param message TaskType message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: ITaskType, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified TaskType message, length delimited. Does not implicitly {@link TaskType.verify|verify} messages.
     * @param message TaskType message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: ITaskType, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a TaskType message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns TaskType
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): TaskType;

    /**
     * Decodes a TaskType message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns TaskType
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): TaskType;

    /**
     * Verifies a TaskType message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a TaskType message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns TaskType
     */
    public static fromObject(object: { [k: string]: any }): TaskType;

    /**
     * Creates a plain object from a TaskType message. Also converts values to other types if specified.
     * @param message TaskType
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: TaskType, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this TaskType to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };
}

export namespace TaskType {

    /** Type enum. */
    enum Type {
        UNKNOWN = 0,
        LIST = 1,
        UPLOAD_GCS = 2
    }
}

/** Properties of a JobRunStatus. */
export interface IJobRunStatus {
}

/** Represents a JobRunStatus. */
export class JobRunStatus implements IJobRunStatus {

    /**
     * Constructs a new JobRunStatus.
     * @param [properties] Properties to set
     */
    constructor(properties?: IJobRunStatus);

    /**
     * Creates a new JobRunStatus instance using the specified properties.
     * @param [properties] Properties to set
     * @returns JobRunStatus instance
     */
    public static create(properties?: IJobRunStatus): JobRunStatus;

    /**
     * Encodes the specified JobRunStatus message. Does not implicitly {@link JobRunStatus.verify|verify} messages.
     * @param message JobRunStatus message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: IJobRunStatus, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified JobRunStatus message, length delimited. Does not implicitly {@link JobRunStatus.verify|verify} messages.
     * @param message JobRunStatus message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: IJobRunStatus, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a JobRunStatus message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns JobRunStatus
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): JobRunStatus;

    /**
     * Decodes a JobRunStatus message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns JobRunStatus
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): JobRunStatus;

    /**
     * Verifies a JobRunStatus message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a JobRunStatus message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns JobRunStatus
     */
    public static fromObject(object: { [k: string]: any }): JobRunStatus;

    /**
     * Creates a plain object from a JobRunStatus message. Also converts values to other types if specified.
     * @param message JobRunStatus
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: JobRunStatus, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this JobRunStatus to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };
}

export namespace JobRunStatus {

    /** Type enum. */
    enum Type {
        NOT_STARTED = 0,
        IN_PROGRESS = 1,
        FAILED = 2,
        SUCCESS = 3
    }
}

/** Properties of a ResourceStatus. */
export interface IResourceStatus {
}

/** Represents a ResourceStatus. */
export class ResourceStatus implements IResourceStatus {

    /**
     * Constructs a new ResourceStatus.
     * @param [properties] Properties to set
     */
    constructor(properties?: IResourceStatus);

    /**
     * Creates a new ResourceStatus instance using the specified properties.
     * @param [properties] Properties to set
     * @returns ResourceStatus instance
     */
    public static create(properties?: IResourceStatus): ResourceStatus;

    /**
     * Encodes the specified ResourceStatus message. Does not implicitly {@link ResourceStatus.verify|verify} messages.
     * @param message ResourceStatus message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encode(message: IResourceStatus, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Encodes the specified ResourceStatus message, length delimited. Does not implicitly {@link ResourceStatus.verify|verify} messages.
     * @param message ResourceStatus message or plain object to encode
     * @param [writer] Writer to encode to
     * @returns Writer
     */
    public static encodeDelimited(message: IResourceStatus, writer?: $protobuf.Writer): $protobuf.Writer;

    /**
     * Decodes a ResourceStatus message from the specified reader or buffer.
     * @param reader Reader or buffer to decode from
     * @param [length] Message length if known beforehand
     * @returns ResourceStatus
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decode(reader: ($protobuf.Reader|Uint8Array), length?: number): ResourceStatus;

    /**
     * Decodes a ResourceStatus message from the specified reader or buffer, length delimited.
     * @param reader Reader or buffer to decode from
     * @returns ResourceStatus
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    public static decodeDelimited(reader: ($protobuf.Reader|Uint8Array)): ResourceStatus;

    /**
     * Verifies a ResourceStatus message.
     * @param message Plain object to verify
     * @returns `null` if valid, otherwise the reason why it is not
     */
    public static verify(message: { [k: string]: any }): (string|null);

    /**
     * Creates a ResourceStatus message from a plain object. Also converts values to their respective internal types.
     * @param object Plain object
     * @returns ResourceStatus
     */
    public static fromObject(object: { [k: string]: any }): ResourceStatus;

    /**
     * Creates a plain object from a ResourceStatus message. Also converts values to other types if specified.
     * @param message ResourceStatus
     * @param [options] Conversion options
     * @returns Plain object
     */
    public static toObject(message: ResourceStatus, options?: $protobuf.IConversionOptions): { [k: string]: any };

    /**
     * Converts this ResourceStatus to JSON.
     * @returns JSON object
     */
    public toJSON(): { [k: string]: any };
}

export namespace ResourceStatus {

    /** Type enum. */
    enum Type {
        TYPE_UNSPECIFIED = 0,
        RUNNING = 1,
        DEPLOYING = 2,
        DELETING = 3,
        FAILED = 4,
        NOT_FOUND = 5,
        UNKNOWN = 6
    }
}
