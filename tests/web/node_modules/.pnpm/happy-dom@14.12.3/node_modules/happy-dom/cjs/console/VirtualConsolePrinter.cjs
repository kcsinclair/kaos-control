"use strict";
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _VirtualConsolePrinter_logEntries, _VirtualConsolePrinter_listeners;
Object.defineProperty(exports, "__esModule", { value: true });
const VirtualConsoleLogLevelEnum_js_1 = __importDefault(require("./enums/VirtualConsoleLogLevelEnum.cjs"));
const Event_js_1 = __importDefault(require("../event/Event.cjs"));
const VirtualConsoleLogEntryStringifier_js_1 = __importDefault(require("./utilities/VirtualConsoleLogEntryStringifier.cjs"));
/**
 * Virtual console printer.
 */
class VirtualConsolePrinter {
    constructor() {
        _VirtualConsolePrinter_logEntries.set(this, []);
        _VirtualConsolePrinter_listeners.set(this, { print: [], clear: [] });
    }
    /**
     * Writes to the output.
     *
     * @param logEntry Log entry.
     */
    print(logEntry) {
        __classPrivateFieldGet(this, _VirtualConsolePrinter_logEntries, "f").push(logEntry);
        this.dispatchEvent(new Event_js_1.default('print'));
    }
    /**
     * Clears the output.
     */
    clear() {
        __classPrivateFieldSet(this, _VirtualConsolePrinter_logEntries, [], "f");
        this.dispatchEvent(new Event_js_1.default('clear'));
    }
    /**
     * Adds an event listener.
     *
     * @param eventType Event type ("print" or "clear").
     * @param listener Listener.
     */
    addEventListener(eventType, listener) {
        if (!__classPrivateFieldGet(this, _VirtualConsolePrinter_listeners, "f")[eventType]) {
            throw new Error(`Event type "${eventType}" is not supported.`);
        }
        __classPrivateFieldGet(this, _VirtualConsolePrinter_listeners, "f")[eventType].push(listener);
    }
    /**
     * Removes an event listener.
     *
     * @param eventType Event type ("print" or "clear").
     * @param listener Listener.
     */
    removeEventListener(eventType, listener) {
        if (!__classPrivateFieldGet(this, _VirtualConsolePrinter_listeners, "f")[eventType]) {
            throw new Error(`Event type "${eventType}" is not supported.`);
        }
        const index = __classPrivateFieldGet(this, _VirtualConsolePrinter_listeners, "f")[eventType].indexOf(listener);
        if (index !== -1) {
            __classPrivateFieldGet(this, _VirtualConsolePrinter_listeners, "f")[eventType].splice(index, 1);
        }
    }
    /**
     * Dispatches an event.
     *
     * @param event Event.
     */
    dispatchEvent(event) {
        if (!__classPrivateFieldGet(this, _VirtualConsolePrinter_listeners, "f")[event.type]) {
            throw new Error(`Event type "${event.type}" is not supported.`);
        }
        for (const listener of __classPrivateFieldGet(this, _VirtualConsolePrinter_listeners, "f")[event.type]) {
            listener(event);
        }
    }
    /**
     * Reads the buffer.
     *
     * @returns Console log entries.
     */
    read() {
        const logEntries = __classPrivateFieldGet(this, _VirtualConsolePrinter_logEntries, "f");
        __classPrivateFieldSet(this, _VirtualConsolePrinter_logEntries, [], "f");
        return logEntries;
    }
    /**
     * Returns the buffer as a string.
     *
     * @param [logLevel] Log level.
     * @returns Buffer as a string of concatenated log entries.
     */
    readAsString(logLevel = VirtualConsoleLogLevelEnum_js_1.default.log) {
        const logEntries = this.read();
        let output = '';
        for (const logEntry of logEntries) {
            if (logEntry.level >= logLevel) {
                output += VirtualConsoleLogEntryStringifier_js_1.default.toString(logEntry);
            }
        }
        return output;
    }
}
_VirtualConsolePrinter_logEntries = new WeakMap(), _VirtualConsolePrinter_listeners = new WeakMap();
exports.default = VirtualConsolePrinter;
//# sourceMappingURL=VirtualConsolePrinter.cjs.map