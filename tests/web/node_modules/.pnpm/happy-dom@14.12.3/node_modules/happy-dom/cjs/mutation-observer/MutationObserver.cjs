"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _MutationObserver_callback, _MutationObserver_listeners, _MutationObserver_window;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const MutationListener_js_1 = __importDefault(require("./MutationListener.cjs"));
/**
 * The MutationObserver interface provides the ability to watch for changes being made to the DOM tree.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationObserver
 */
class MutationObserver {
    /**
     * Constructor.
     *
     * @param callback Callback.
     */
    constructor(callback) {
        _MutationObserver_callback.set(this, void 0);
        _MutationObserver_listeners.set(this, []);
        _MutationObserver_window.set(this, null);
        __classPrivateFieldSet(this, _MutationObserver_callback, callback, "f");
    }
    /**
     * Starts observing.
     *
     * @param target Target.
     * @param options Options.
     */
    observe(target, options) {
        if (!target) {
            throw new TypeError(`Failed to execute 'observe' on 'MutationObserver': The first parameter "target" should be of type "Node".`);
        }
        if (options && (options.attributeFilter || options.attributeOldValue)) {
            if (options.attributes === undefined) {
                options = Object.assign({}, options, {
                    attributes: true,
                    attributeFilter: options.attributeFilter,
                    attributeOldValue: options.attributeOldValue
                });
            }
            if (!options.attributes && options.attributeOldValue) {
                throw new TypeError(`Failed to execute 'observe' on 'MutationObserver': The options object may only set 'attributeOldValue' to true when 'attributes' is true or not present.`);
            }
            if (!options.attributes && options.attributeFilter) {
                throw new TypeError(`Failed to execute 'observe' on 'MutationObserver': The options object may only set 'attributeFilter' when 'attributes' is true or not present.`);
            }
        }
        if (options && options.characterDataOldValue) {
            if (options.characterData === undefined) {
                options = Object.assign({}, options, {
                    characterData: true,
                    characterDataOldValue: options.characterDataOldValue
                });
            }
            if (!options.characterData && options.characterDataOldValue) {
                throw new TypeError(`Failed to execute 'observe' on 'MutationObserver': The options object may only set 'characterDataOldValue' to true when 'characterData' is true or not present.`);
            }
        }
        if (!options || (!options.childList && !options.attributes && !options.characterData)) {
            throw new TypeError(`Failed to execute 'observe' on 'MutationObserver': The options object must set at least one of 'attributes', 'characterData', or 'childList' to true.`);
        }
        if (!__classPrivateFieldGet(this, _MutationObserver_window, "f")) {
            __classPrivateFieldSet(this, _MutationObserver_window, target[PropertySymbol.ownerDocument]
                ? target[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow]
                : target[PropertySymbol.ownerWindow], "f");
        }
        // Makes sure that attribute names are lower case.
        // TODO: Is this correct?
        options = Object.assign({}, options, {
            attributeFilter: options.attributeFilter
                ? options.attributeFilter.map((name) => name.toLowerCase())
                : null
        });
        /**
         * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationObserver/observe#reusing_mutationobservers
         */
        for (const listener of __classPrivateFieldGet(this, _MutationObserver_listeners, "f")) {
            if (listener.target === target) {
                listener.options = options;
                return;
            }
        }
        const listener = new MutationListener_js_1.default({
            window: __classPrivateFieldGet(this, _MutationObserver_window, "f"),
            options,
            callback: __classPrivateFieldGet(this, _MutationObserver_callback, "f").bind(this),
            observer: this,
            target
        });
        __classPrivateFieldGet(this, _MutationObserver_listeners, "f").push(listener);
        // Stores all observers on the window object, so that they can be disconnected when the window is closed.
        __classPrivateFieldGet(this, _MutationObserver_window, "f")[PropertySymbol.mutationObservers].push(this);
        // Starts observing target node.
        target[PropertySymbol.observe](listener);
    }
    /**
     * Disconnects.
     */
    disconnect() {
        if (__classPrivateFieldGet(this, _MutationObserver_listeners, "f").length === 0) {
            return;
        }
        const mutationObservers = __classPrivateFieldGet(this, _MutationObserver_window, "f")[PropertySymbol.mutationObservers];
        const index = mutationObservers.indexOf(this);
        if (index !== -1) {
            mutationObservers.splice(index, 1);
        }
        for (const listener of __classPrivateFieldGet(this, _MutationObserver_listeners, "f")) {
            listener.target[PropertySymbol.unobserve](listener);
            listener.destroy();
        }
        __classPrivateFieldSet(this, _MutationObserver_listeners, [], "f");
    }
    /**
     * Returns a list of all matching DOM changes that have been detected but not yet processed by the observer's callback function, leaving the mutation queue empty.
     *
     * @returns Records.
     */
    takeRecords() {
        let records = [];
        for (const listener of __classPrivateFieldGet(this, _MutationObserver_listeners, "f")) {
            records = records.concat(listener.takeRecords());
        }
        return records;
    }
}
_MutationObserver_callback = new WeakMap(), _MutationObserver_listeners = new WeakMap(), _MutationObserver_window = new WeakMap();
exports.default = MutationObserver;
//# sourceMappingURL=MutationObserver.cjs.map