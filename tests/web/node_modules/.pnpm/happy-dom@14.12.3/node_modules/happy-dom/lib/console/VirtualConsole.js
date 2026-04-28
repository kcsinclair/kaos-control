var __classPrivateFieldSet = (this && this.__classPrivateFieldSet) || function (receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return (kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value)), value;
};
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _VirtualConsole_printer, _VirtualConsole_count, _VirtualConsole_time, _VirtualConsole_groupID, _VirtualConsole_groups;
import VirtualConsoleLogLevelEnum from './enums/VirtualConsoleLogLevelEnum.js';
import VirtualConsoleLogTypeEnum from './enums/VirtualConsoleLogTypeEnum.js';
/**
 * Virtual Console.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Console
 */
class VirtualConsole {
    /**
     * Constructor.
     *
     * @param printer Console printer.
     */
    constructor(printer) {
        _VirtualConsole_printer.set(this, void 0);
        _VirtualConsole_count.set(this, {});
        _VirtualConsole_time.set(this, {});
        _VirtualConsole_groupID.set(this, 0);
        _VirtualConsole_groups.set(this, []);
        __classPrivateFieldSet(this, _VirtualConsole_printer, printer, "f");
    }
    /**
     * Writes an error message to the console if the assertion is false. If the assertion is true, nothing happens.
     *
     * @param assertion Assertion.
     * @param args Arguments.
     */
    assert(assertion, ...args) {
        if (!assertion) {
            __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
                type: VirtualConsoleLogTypeEnum.assert,
                level: VirtualConsoleLogLevelEnum.error,
                message: ['Assertion failed:', ...args],
                group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
            });
        }
    }
    /**
     * Clears the console.
     */
    clear() {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").clear();
    }
    /**
     * Logs the number of times that this particular call to count() has been called.
     *
     * @param [label='default'] Label.
     */
    count(label = 'default') {
        if (!__classPrivateFieldGet(this, _VirtualConsole_count, "f")[label]) {
            __classPrivateFieldGet(this, _VirtualConsole_count, "f")[label] = 0;
        }
        __classPrivateFieldGet(this, _VirtualConsole_count, "f")[label]++;
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.count,
            level: VirtualConsoleLogLevelEnum.info,
            message: [`${label}: ${__classPrivateFieldGet(this, _VirtualConsole_count, "f")[label]}`],
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Resets the counter.
     *
     * @param [label='default'] Label.
     */
    countReset(label = 'default') {
        delete __classPrivateFieldGet(this, _VirtualConsole_count, "f")[label];
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.countReset,
            level: VirtualConsoleLogLevelEnum.warn,
            message: [`${label}: 0`],
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Outputs a message to the web console at the "debug" log level.
     *
     * @param args Arguments.
     */
    debug(...args) {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.debug,
            level: VirtualConsoleLogLevelEnum.log,
            message: args,
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Displays an interactive list of the properties of the specified JavaScript object.
     *
     * @param data Data.
     */
    dir(data) {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.dir,
            level: VirtualConsoleLogLevelEnum.log,
            message: [data],
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Displays an interactive tree of the descendant elements of the specified XML/HTML element.
     *
     * @param data Data.
     */
    dirxml(data) {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.dirxml,
            level: VirtualConsoleLogLevelEnum.log,
            message: [data],
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Outputs an error message to the console.
     *
     * @param args Arguments.
     */
    error(...args) {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.error,
            level: VirtualConsoleLogLevelEnum.error,
            message: args,
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Alias for error().
     *
     * @deprecated
     * @alias error()
     * @param args Arguments.
     */
    exception(...args) {
        this.error(...args);
    }
    /**
     * Creates a new inline group in the console, causing any subsequent console messages to be indented by an additional level, until console.groupEnd() is called.
     *
     * @param [label] Label.
     */
    group(label) {
        var _a;
        __classPrivateFieldSet(this, _VirtualConsole_groupID, (_a = __classPrivateFieldGet(this, _VirtualConsole_groupID, "f"), _a++, _a), "f");
        const group = {
            id: __classPrivateFieldGet(this, _VirtualConsole_groupID, "f"),
            label: label || 'default',
            collapsed: false,
            parent: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        };
        __classPrivateFieldGet(this, _VirtualConsole_groups, "f").push(group);
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.group,
            level: VirtualConsoleLogLevelEnum.log,
            message: [label || 'default'],
            group
        });
    }
    /**
     * Creates a new inline group in the console, but prints it as collapsed, requiring the use of a disclosure button to expand it.
     *
     * @param [label] Label.
     */
    groupCollapsed(label) {
        var _a;
        __classPrivateFieldSet(this, _VirtualConsole_groupID, (_a = __classPrivateFieldGet(this, _VirtualConsole_groupID, "f"), _a++, _a), "f");
        const group = {
            id: __classPrivateFieldGet(this, _VirtualConsole_groupID, "f"),
            label: label || 'default',
            collapsed: true,
            parent: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        };
        __classPrivateFieldGet(this, _VirtualConsole_groups, "f").push(group);
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.groupCollapsed,
            level: VirtualConsoleLogLevelEnum.log,
            message: [label || 'default'],
            group
        });
    }
    /**
     * Exits the current inline group in the console.
     */
    groupEnd() {
        if (__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length === 0) {
            return;
        }
        __classPrivateFieldGet(this, _VirtualConsole_groups, "f").pop();
    }
    /**
     *
     * @param args
     */
    info(...args) {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.info,
            level: VirtualConsoleLogLevelEnum.info,
            message: args,
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Outputs a message to the console.
     *
     * @param args Arguments.
     */
    log(...args) {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.log,
            level: VirtualConsoleLogLevelEnum.log,
            message: args,
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Starts recording a performance profile.
     *
     * TODO: Implement this.
     */
    profile() {
        throw new Error('Method not implemented.');
    }
    /**
     * Stops recording a performance profile.
     *
     * TODO: Implement this.
     */
    profileEnd() {
        throw new Error('Method not implemented.');
    }
    /**
     * Displays tabular data as a table.
     *
     * @param data Data.
     */
    table(data) {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.table,
            level: VirtualConsoleLogLevelEnum.log,
            message: [data],
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Starts a timer you can use to track how long an operation takes.
     *
     * @param [label=default] Label.
     */
    time(label = 'default') {
        __classPrivateFieldGet(this, _VirtualConsole_time, "f")[label] = performance.now();
    }
    /**
     * Stops a timer that was previously started by calling console.time().
     * The method logs the elapsed time in milliseconds.
     *
     * @param [label=default] Label.
     */
    timeEnd(label = 'default') {
        const time = __classPrivateFieldGet(this, _VirtualConsole_time, "f")[label];
        if (time) {
            const duration = performance.now() - time;
            __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
                type: VirtualConsoleLogTypeEnum.timeEnd,
                level: VirtualConsoleLogLevelEnum.info,
                message: [`${label}: ${duration}ms - timer ended`],
                group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
            });
        }
    }
    /**
     * Logs the current value of a timer that was previously started by calling console.time().
     * The method logs the elapsed time in milliseconds.
     *
     * @param [label=default] Label.
     * @param [args] Arguments.
     */
    timeLog(label = 'default', ...args) {
        const time = __classPrivateFieldGet(this, _VirtualConsole_time, "f")[label];
        if (time) {
            const duration = performance.now() - time;
            __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
                type: VirtualConsoleLogTypeEnum.timeLog,
                level: VirtualConsoleLogLevelEnum.info,
                message: [`${label}: ${duration}ms`, ...args],
                group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
            });
        }
    }
    /**
     * Adds a single marker to the browser's Performance tool.
     *
     * TODO: Implement this.
     */
    timeStamp() {
        throw new Error('Method not implemented.');
    }
    /**
     * Outputs a stack trace to the console.
     *
     * @param args Arguments.
     */
    trace(...args) {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.trace,
            level: VirtualConsoleLogLevelEnum.log,
            message: [...args, new Error('stack').stack.replace('Error: stack', '')],
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
    /**
     * Outputs a warning message to the console.
     *
     * @param args Arguments.
     */
    warn(...args) {
        __classPrivateFieldGet(this, _VirtualConsole_printer, "f").print({
            type: VirtualConsoleLogTypeEnum.warn,
            level: VirtualConsoleLogLevelEnum.warn,
            message: args,
            group: __classPrivateFieldGet(this, _VirtualConsole_groups, "f")[__classPrivateFieldGet(this, _VirtualConsole_groups, "f").length - 1] || null
        });
    }
}
_VirtualConsole_printer = new WeakMap(), _VirtualConsole_count = new WeakMap(), _VirtualConsole_time = new WeakMap(), _VirtualConsole_groupID = new WeakMap(), _VirtualConsole_groups = new WeakMap();
export default VirtualConsole;
//# sourceMappingURL=VirtualConsole.js.map