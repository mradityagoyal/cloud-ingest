/*eslint-disable block-scoped-var, no-redeclare, no-control-regex, no-prototype-builtins*/
"use strict";

var $protobuf = require("protobufjs/minimal");

// Common aliases
var $Reader = $protobuf.Reader, $Writer = $protobuf.Writer, $util = $protobuf.util;

// Exported root namespace
var $root = $protobuf.roots["default"] || ($protobuf.roots["default"] = {});

$root.TaskFailureType = (function() {

    /**
     * Properties of a TaskFailureType.
     * @exports ITaskFailureType
     * @interface ITaskFailureType
     */

    /**
     * Constructs a new TaskFailureType.
     * @exports TaskFailureType
     * @classdesc Represents a TaskFailureType.
     * @constructor
     * @param {ITaskFailureType=} [properties] Properties to set
     */
    function TaskFailureType(properties) {
        if (properties)
            for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                if (properties[keys[i]] != null)
                    this[keys[i]] = properties[keys[i]];
    }

    /**
     * Creates a new TaskFailureType instance using the specified properties.
     * @function create
     * @memberof TaskFailureType
     * @static
     * @param {ITaskFailureType=} [properties] Properties to set
     * @returns {TaskFailureType} TaskFailureType instance
     */
    TaskFailureType.create = function create(properties) {
        return new TaskFailureType(properties);
    };

    /**
     * Encodes the specified TaskFailureType message. Does not implicitly {@link TaskFailureType.verify|verify} messages.
     * @function encode
     * @memberof TaskFailureType
     * @static
     * @param {ITaskFailureType} message TaskFailureType message or plain object to encode
     * @param {$protobuf.Writer} [writer] Writer to encode to
     * @returns {$protobuf.Writer} Writer
     */
    TaskFailureType.encode = function encode(message, writer) {
        if (!writer)
            writer = $Writer.create();
        return writer;
    };

    /**
     * Encodes the specified TaskFailureType message, length delimited. Does not implicitly {@link TaskFailureType.verify|verify} messages.
     * @function encodeDelimited
     * @memberof TaskFailureType
     * @static
     * @param {ITaskFailureType} message TaskFailureType message or plain object to encode
     * @param {$protobuf.Writer} [writer] Writer to encode to
     * @returns {$protobuf.Writer} Writer
     */
    TaskFailureType.encodeDelimited = function encodeDelimited(message, writer) {
        return this.encode(message, writer).ldelim();
    };

    /**
     * Decodes a TaskFailureType message from the specified reader or buffer.
     * @function decode
     * @memberof TaskFailureType
     * @static
     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
     * @param {number} [length] Message length if known beforehand
     * @returns {TaskFailureType} TaskFailureType
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    TaskFailureType.decode = function decode(reader, length) {
        if (!(reader instanceof $Reader))
            reader = $Reader.create(reader);
        var end = length === undefined ? reader.len : reader.pos + length, message = new $root.TaskFailureType();
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
            default:
                reader.skipType(tag & 7);
                break;
            }
        }
        return message;
    };

    /**
     * Decodes a TaskFailureType message from the specified reader or buffer, length delimited.
     * @function decodeDelimited
     * @memberof TaskFailureType
     * @static
     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
     * @returns {TaskFailureType} TaskFailureType
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    TaskFailureType.decodeDelimited = function decodeDelimited(reader) {
        if (!(reader instanceof $Reader))
            reader = new $Reader(reader);
        return this.decode(reader, reader.uint32());
    };

    /**
     * Verifies a TaskFailureType message.
     * @function verify
     * @memberof TaskFailureType
     * @static
     * @param {Object.<string,*>} message Plain object to verify
     * @returns {string|null} `null` if valid, otherwise the reason why it is not
     */
    TaskFailureType.verify = function verify(message) {
        if (typeof message !== "object" || message === null)
            return "object expected";
        return null;
    };

    /**
     * Creates a TaskFailureType message from a plain object. Also converts values to their respective internal types.
     * @function fromObject
     * @memberof TaskFailureType
     * @static
     * @param {Object.<string,*>} object Plain object
     * @returns {TaskFailureType} TaskFailureType
     */
    TaskFailureType.fromObject = function fromObject(object) {
        if (object instanceof $root.TaskFailureType)
            return object;
        return new $root.TaskFailureType();
    };

    /**
     * Creates a plain object from a TaskFailureType message. Also converts values to other types if specified.
     * @function toObject
     * @memberof TaskFailureType
     * @static
     * @param {TaskFailureType} message TaskFailureType
     * @param {$protobuf.IConversionOptions} [options] Conversion options
     * @returns {Object.<string,*>} Plain object
     */
    TaskFailureType.toObject = function toObject() {
        return {};
    };

    /**
     * Converts this TaskFailureType to JSON.
     * @function toJSON
     * @memberof TaskFailureType
     * @instance
     * @returns {Object.<string,*>} JSON object
     */
    TaskFailureType.prototype.toJSON = function toJSON() {
        return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
    };

    /**
     * Type enum.
     * @enum {string}
     * @property {number} UNUSED=0 UNUSED value
     * @property {number} UNKNOWN=1 UNKNOWN value
     * @property {number} FILE_MODIFIED_FAILURE=2 FILE_MODIFIED_FAILURE value
     * @property {number} MD5_MISMATCH_FAILURE=3 MD5_MISMATCH_FAILURE value
     * @property {number} PRECONDITION_FAILURE=4 PRECONDITION_FAILURE value
     * @property {number} FILE_NOT_FOUND_FAILURE=5 FILE_NOT_FOUND_FAILURE value
     * @property {number} PERMISSION_FAILURE=6 PERMISSION_FAILURE value
     */
    TaskFailureType.Type = (function() {
        var valuesById = {}, values = Object.create(valuesById);
        values[valuesById[0] = "UNUSED"] = 0;
        values[valuesById[1] = "UNKNOWN"] = 1;
        values[valuesById[2] = "FILE_MODIFIED_FAILURE"] = 2;
        values[valuesById[3] = "MD5_MISMATCH_FAILURE"] = 3;
        values[valuesById[4] = "PRECONDITION_FAILURE"] = 4;
        values[valuesById[5] = "FILE_NOT_FOUND_FAILURE"] = 5;
        values[valuesById[6] = "PERMISSION_FAILURE"] = 6;
        return values;
    })();

    return TaskFailureType;
})();

$root.TaskStatus = (function() {

    /**
     * Properties of a TaskStatus.
     * @exports ITaskStatus
     * @interface ITaskStatus
     */

    /**
     * Constructs a new TaskStatus.
     * @exports TaskStatus
     * @classdesc Represents a TaskStatus.
     * @constructor
     * @param {ITaskStatus=} [properties] Properties to set
     */
    function TaskStatus(properties) {
        if (properties)
            for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                if (properties[keys[i]] != null)
                    this[keys[i]] = properties[keys[i]];
    }

    /**
     * Creates a new TaskStatus instance using the specified properties.
     * @function create
     * @memberof TaskStatus
     * @static
     * @param {ITaskStatus=} [properties] Properties to set
     * @returns {TaskStatus} TaskStatus instance
     */
    TaskStatus.create = function create(properties) {
        return new TaskStatus(properties);
    };

    /**
     * Encodes the specified TaskStatus message. Does not implicitly {@link TaskStatus.verify|verify} messages.
     * @function encode
     * @memberof TaskStatus
     * @static
     * @param {ITaskStatus} message TaskStatus message or plain object to encode
     * @param {$protobuf.Writer} [writer] Writer to encode to
     * @returns {$protobuf.Writer} Writer
     */
    TaskStatus.encode = function encode(message, writer) {
        if (!writer)
            writer = $Writer.create();
        return writer;
    };

    /**
     * Encodes the specified TaskStatus message, length delimited. Does not implicitly {@link TaskStatus.verify|verify} messages.
     * @function encodeDelimited
     * @memberof TaskStatus
     * @static
     * @param {ITaskStatus} message TaskStatus message or plain object to encode
     * @param {$protobuf.Writer} [writer] Writer to encode to
     * @returns {$protobuf.Writer} Writer
     */
    TaskStatus.encodeDelimited = function encodeDelimited(message, writer) {
        return this.encode(message, writer).ldelim();
    };

    /**
     * Decodes a TaskStatus message from the specified reader or buffer.
     * @function decode
     * @memberof TaskStatus
     * @static
     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
     * @param {number} [length] Message length if known beforehand
     * @returns {TaskStatus} TaskStatus
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    TaskStatus.decode = function decode(reader, length) {
        if (!(reader instanceof $Reader))
            reader = $Reader.create(reader);
        var end = length === undefined ? reader.len : reader.pos + length, message = new $root.TaskStatus();
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
            default:
                reader.skipType(tag & 7);
                break;
            }
        }
        return message;
    };

    /**
     * Decodes a TaskStatus message from the specified reader or buffer, length delimited.
     * @function decodeDelimited
     * @memberof TaskStatus
     * @static
     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
     * @returns {TaskStatus} TaskStatus
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    TaskStatus.decodeDelimited = function decodeDelimited(reader) {
        if (!(reader instanceof $Reader))
            reader = new $Reader(reader);
        return this.decode(reader, reader.uint32());
    };

    /**
     * Verifies a TaskStatus message.
     * @function verify
     * @memberof TaskStatus
     * @static
     * @param {Object.<string,*>} message Plain object to verify
     * @returns {string|null} `null` if valid, otherwise the reason why it is not
     */
    TaskStatus.verify = function verify(message) {
        if (typeof message !== "object" || message === null)
            return "object expected";
        return null;
    };

    /**
     * Creates a TaskStatus message from a plain object. Also converts values to their respective internal types.
     * @function fromObject
     * @memberof TaskStatus
     * @static
     * @param {Object.<string,*>} object Plain object
     * @returns {TaskStatus} TaskStatus
     */
    TaskStatus.fromObject = function fromObject(object) {
        if (object instanceof $root.TaskStatus)
            return object;
        return new $root.TaskStatus();
    };

    /**
     * Creates a plain object from a TaskStatus message. Also converts values to other types if specified.
     * @function toObject
     * @memberof TaskStatus
     * @static
     * @param {TaskStatus} message TaskStatus
     * @param {$protobuf.IConversionOptions} [options] Conversion options
     * @returns {Object.<string,*>} Plain object
     */
    TaskStatus.toObject = function toObject() {
        return {};
    };

    /**
     * Converts this TaskStatus to JSON.
     * @function toJSON
     * @memberof TaskStatus
     * @instance
     * @returns {Object.<string,*>} JSON object
     */
    TaskStatus.prototype.toJSON = function toJSON() {
        return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
    };

    /**
     * Type enum.
     * @enum {string}
     * @property {number} UNQUEUED=0 UNQUEUED value
     * @property {number} QUEUED=1 QUEUED value
     * @property {number} FAILED=2 FAILED value
     * @property {number} SUCCESS=3 SUCCESS value
     */
    TaskStatus.Type = (function() {
        var valuesById = {}, values = Object.create(valuesById);
        values[valuesById[0] = "UNQUEUED"] = 0;
        values[valuesById[1] = "QUEUED"] = 1;
        values[valuesById[2] = "FAILED"] = 2;
        values[valuesById[3] = "SUCCESS"] = 3;
        return values;
    })();

    return TaskStatus;
})();

$root.TaskType = (function() {

    /**
     * Properties of a TaskType.
     * @exports ITaskType
     * @interface ITaskType
     */

    /**
     * Constructs a new TaskType.
     * @exports TaskType
     * @classdesc Represents a TaskType.
     * @constructor
     * @param {ITaskType=} [properties] Properties to set
     */
    function TaskType(properties) {
        if (properties)
            for (var keys = Object.keys(properties), i = 0; i < keys.length; ++i)
                if (properties[keys[i]] != null)
                    this[keys[i]] = properties[keys[i]];
    }

    /**
     * Creates a new TaskType instance using the specified properties.
     * @function create
     * @memberof TaskType
     * @static
     * @param {ITaskType=} [properties] Properties to set
     * @returns {TaskType} TaskType instance
     */
    TaskType.create = function create(properties) {
        return new TaskType(properties);
    };

    /**
     * Encodes the specified TaskType message. Does not implicitly {@link TaskType.verify|verify} messages.
     * @function encode
     * @memberof TaskType
     * @static
     * @param {ITaskType} message TaskType message or plain object to encode
     * @param {$protobuf.Writer} [writer] Writer to encode to
     * @returns {$protobuf.Writer} Writer
     */
    TaskType.encode = function encode(message, writer) {
        if (!writer)
            writer = $Writer.create();
        return writer;
    };

    /**
     * Encodes the specified TaskType message, length delimited. Does not implicitly {@link TaskType.verify|verify} messages.
     * @function encodeDelimited
     * @memberof TaskType
     * @static
     * @param {ITaskType} message TaskType message or plain object to encode
     * @param {$protobuf.Writer} [writer] Writer to encode to
     * @returns {$protobuf.Writer} Writer
     */
    TaskType.encodeDelimited = function encodeDelimited(message, writer) {
        return this.encode(message, writer).ldelim();
    };

    /**
     * Decodes a TaskType message from the specified reader or buffer.
     * @function decode
     * @memberof TaskType
     * @static
     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
     * @param {number} [length] Message length if known beforehand
     * @returns {TaskType} TaskType
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    TaskType.decode = function decode(reader, length) {
        if (!(reader instanceof $Reader))
            reader = $Reader.create(reader);
        var end = length === undefined ? reader.len : reader.pos + length, message = new $root.TaskType();
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
            default:
                reader.skipType(tag & 7);
                break;
            }
        }
        return message;
    };

    /**
     * Decodes a TaskType message from the specified reader or buffer, length delimited.
     * @function decodeDelimited
     * @memberof TaskType
     * @static
     * @param {$protobuf.Reader|Uint8Array} reader Reader or buffer to decode from
     * @returns {TaskType} TaskType
     * @throws {Error} If the payload is not a reader or valid buffer
     * @throws {$protobuf.util.ProtocolError} If required fields are missing
     */
    TaskType.decodeDelimited = function decodeDelimited(reader) {
        if (!(reader instanceof $Reader))
            reader = new $Reader(reader);
        return this.decode(reader, reader.uint32());
    };

    /**
     * Verifies a TaskType message.
     * @function verify
     * @memberof TaskType
     * @static
     * @param {Object.<string,*>} message Plain object to verify
     * @returns {string|null} `null` if valid, otherwise the reason why it is not
     */
    TaskType.verify = function verify(message) {
        if (typeof message !== "object" || message === null)
            return "object expected";
        return null;
    };

    /**
     * Creates a TaskType message from a plain object. Also converts values to their respective internal types.
     * @function fromObject
     * @memberof TaskType
     * @static
     * @param {Object.<string,*>} object Plain object
     * @returns {TaskType} TaskType
     */
    TaskType.fromObject = function fromObject(object) {
        if (object instanceof $root.TaskType)
            return object;
        return new $root.TaskType();
    };

    /**
     * Creates a plain object from a TaskType message. Also converts values to other types if specified.
     * @function toObject
     * @memberof TaskType
     * @static
     * @param {TaskType} message TaskType
     * @param {$protobuf.IConversionOptions} [options] Conversion options
     * @returns {Object.<string,*>} Plain object
     */
    TaskType.toObject = function toObject() {
        return {};
    };

    /**
     * Converts this TaskType to JSON.
     * @function toJSON
     * @memberof TaskType
     * @instance
     * @returns {Object.<string,*>} JSON object
     */
    TaskType.prototype.toJSON = function toJSON() {
        return this.constructor.toObject(this, $protobuf.util.toJSONOptions);
    };

    /**
     * Type enum.
     * @enum {string}
     * @property {number} UNKNOWN=0 UNKNOWN value
     * @property {number} LIST=1 LIST value
     * @property {number} UPLOAD_GCS=2 UPLOAD_GCS value
     * @property {number} LOAD_BQ=3 LOAD_BQ value
     */
    TaskType.Type = (function() {
        var valuesById = {}, values = Object.create(valuesById);
        values[valuesById[0] = "UNKNOWN"] = 0;
        values[valuesById[1] = "LIST"] = 1;
        values[valuesById[2] = "UPLOAD_GCS"] = 2;
        values[valuesById[3] = "LOAD_BQ"] = 3;
        return values;
    })();

    return TaskType;
})();

module.exports = $root;
