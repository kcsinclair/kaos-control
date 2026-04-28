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
var _MutationListener_window, _MutationListener_observer, _MutationListener_callback, _MutationListener_records, _MutationListener_immediate;
/**
 * Mutation Observer Listener.
 */
class MutationListener {
    /**
     * Constructor.
     *
     * @param init Options.
     * @param init.window Window.
     * @param init.options Options.
     * @param init.target Target.
     * @param init.observer Observer.
     * @param init.callback Callback.
     */
    constructor(init) {
        _MutationListener_window.set(this, void 0);
        _MutationListener_observer.set(this, void 0);
        _MutationListener_callback.set(this, void 0);
        _MutationListener_records.set(this, []);
        _MutationListener_immediate.set(this, null);
        this.options = init.options;
        this.target = init.target;
        __classPrivateFieldSet(this, _MutationListener_window, init.window, "f");
        __classPrivateFieldSet(this, _MutationListener_observer, init.observer, "f");
        __classPrivateFieldSet(this, _MutationListener_callback, init.callback, "f");
    }
    /**
     * Reports mutations.
     *
     * @param record Record.
     */
    report(record) {
        if (!__classPrivateFieldGet(this, _MutationListener_records, "f")) {
            return;
        }
        __classPrivateFieldGet(this, _MutationListener_records, "f").push(record);
        if (__classPrivateFieldGet(this, _MutationListener_immediate, "f")) {
            __classPrivateFieldGet(this, _MutationListener_window, "f").cancelAnimationFrame(__classPrivateFieldGet(this, _MutationListener_immediate, "f"));
        }
        __classPrivateFieldSet(this, _MutationListener_immediate, __classPrivateFieldGet(this, _MutationListener_window, "f").requestAnimationFrame(() => {
            const records = __classPrivateFieldGet(this, _MutationListener_records, "f");
            if (records?.length > 0) {
                __classPrivateFieldSet(this, _MutationListener_records, [], "f");
                __classPrivateFieldGet(this, _MutationListener_callback, "f").call(this, records, __classPrivateFieldGet(this, _MutationListener_observer, "f"));
            }
        }), "f");
    }
    /**
     * Destroys the listener.
     */
    takeRecords() {
        if (__classPrivateFieldGet(this, _MutationListener_immediate, "f")) {
            __classPrivateFieldGet(this, _MutationListener_window, "f").cancelAnimationFrame(__classPrivateFieldGet(this, _MutationListener_immediate, "f"));
        }
        const records = __classPrivateFieldGet(this, _MutationListener_records, "f");
        __classPrivateFieldSet(this, _MutationListener_records, [], "f");
        return records;
    }
    /**
     * Destroys the listener.
     */
    destroy() {
        if (__classPrivateFieldGet(this, _MutationListener_immediate, "f")) {
            __classPrivateFieldGet(this, _MutationListener_window, "f").cancelAnimationFrame(__classPrivateFieldGet(this, _MutationListener_immediate, "f"));
        }
        this.options = null;
        this.target = null;
        __classPrivateFieldSet(this, _MutationListener_observer, null, "f");
        __classPrivateFieldSet(this, _MutationListener_callback, null, "f");
        __classPrivateFieldSet(this, _MutationListener_immediate, null, "f");
        __classPrivateFieldSet(this, _MutationListener_records, null, "f");
    }
}
_MutationListener_window = new WeakMap(), _MutationListener_observer = new WeakMap(), _MutationListener_callback = new WeakMap(), _MutationListener_records = new WeakMap(), _MutationListener_immediate = new WeakMap();
export default MutationListener;
//# sourceMappingURL=MutationListener.js.map