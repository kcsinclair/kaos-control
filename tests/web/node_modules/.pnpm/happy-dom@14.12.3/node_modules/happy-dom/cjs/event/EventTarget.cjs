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
var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
var _EventTarget_instances, _EventTarget_getWindow, _a, _b;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const EventPhaseEnum_js_1 = __importDefault(require("./EventPhaseEnum.cjs"));
const WindowErrorUtility_js_1 = __importDefault(require("../window/WindowErrorUtility.cjs"));
const WindowBrowserSettingsReader_js_1 = __importDefault(require("../window/WindowBrowserSettingsReader.cjs"));
const BrowserErrorCaptureEnum_js_1 = __importDefault(require("../browser/enums/BrowserErrorCaptureEnum.cjs"));
/**
 * Handles events.
 */
class EventTarget {
    constructor() {
        _EventTarget_instances.add(this);
        this[_a] = {};
        this[_b] = {};
    }
    /**
     * Return a default description for the EventTarget class.
     */
    get [(_EventTarget_instances = new WeakSet(), _a = PropertySymbol.listeners, _b = PropertySymbol.listenerOptions, Symbol.toStringTag)]() {
        return 'EventTarget';
    }
    /**
     * Adds an event listener.
     *
     * @param type Event type.
     * @param listener Listener.
     * @param options An object that specifies characteristics about the event listener.(currently only once)
     * @param options.once
     */
    addEventListener(type, listener, options) {
        const listenerOptions = typeof options === 'boolean' ? { capture: options } : options || null;
        this[PropertySymbol.listeners][type] = this[PropertySymbol.listeners][type] || [];
        this[PropertySymbol.listenerOptions][type] = this[PropertySymbol.listenerOptions][type] || [];
        if (this[PropertySymbol.listeners][type].includes(listener)) {
            return;
        }
        this[PropertySymbol.listeners][type].push(listener);
        this[PropertySymbol.listenerOptions][type].push(listenerOptions);
        // Tracks the amount of capture event listeners to improve performance when they are not used.
        if (listenerOptions && listenerOptions.capture) {
            const window = __classPrivateFieldGet(this, _EventTarget_instances, "m", _EventTarget_getWindow).call(this);
            if (window) {
                window[PropertySymbol.captureEventListenerCount][type] =
                    window[PropertySymbol.captureEventListenerCount][type] ?? 0;
                window[PropertySymbol.captureEventListenerCount][type]++;
            }
        }
    }
    /**
     * Adds an event listener.
     *
     * @param type Event type.
     * @param listener Listener.
     */
    removeEventListener(type, listener) {
        if (this[PropertySymbol.listeners][type]) {
            const index = this[PropertySymbol.listeners][type].indexOf(listener);
            if (index !== -1) {
                // Tracks the amount of capture event listeners to improve performance when they are not used.
                if (this[PropertySymbol.listenerOptions][type][index] &&
                    this[PropertySymbol.listenerOptions][type][index].capture) {
                    const window = __classPrivateFieldGet(this, _EventTarget_instances, "m", _EventTarget_getWindow).call(this);
                    if (window && window[PropertySymbol.captureEventListenerCount][type]) {
                        window[PropertySymbol.captureEventListenerCount][type]--;
                    }
                }
                this[PropertySymbol.listeners][type].splice(index, 1);
                this[PropertySymbol.listenerOptions][type].splice(index, 1);
            }
        }
    }
    /**
     * Dispatches an event.
     *
     * @see https://www.w3.org/TR/DOM-Level-3-Events/#event-flow
     * @see https://www.quirksmode.org/js/events_order.html#link4
     * @param event Event.
     * @returns The return value is false if event is cancelable and at least one of the event handlers which handled this event called Event.preventDefault().
     */
    dispatchEvent(event) {
        const window = __classPrivateFieldGet(this, _EventTarget_instances, "m", _EventTarget_getWindow).call(this);
        if (event.eventPhase === EventPhaseEnum_js_1.default.none) {
            event[PropertySymbol.target] = this;
            const composedPath = event.composedPath();
            // Capturing phase
            // We only need to iterate over the composed path if there are capture event listeners.
            if (window && window[PropertySymbol.captureEventListenerCount][event.type]) {
                event.eventPhase = EventPhaseEnum_js_1.default.capturing;
                for (let i = composedPath.length - 1; i >= 0; i--) {
                    composedPath[i].dispatchEvent(event);
                    if (event[PropertySymbol.propagationStopped] ||
                        event[PropertySymbol.immediatePropagationStopped]) {
                        break;
                    }
                }
            }
            // At target phase
            event.eventPhase = EventPhaseEnum_js_1.default.atTarget;
            this.dispatchEvent(event);
            // Bubbling phase
            if (event.bubbles &&
                !event[PropertySymbol.propagationStopped] &&
                !event[PropertySymbol.immediatePropagationStopped]) {
                event.eventPhase = EventPhaseEnum_js_1.default.bubbling;
                for (let i = 1; i < composedPath.length; i++) {
                    composedPath[i].dispatchEvent(event);
                    if (event[PropertySymbol.propagationStopped] ||
                        event[PropertySymbol.immediatePropagationStopped]) {
                        break;
                    }
                }
            }
            // None phase (completed)
            event.eventPhase = EventPhaseEnum_js_1.default.none;
            return !(event.cancelable && event.defaultPrevented);
        }
        event[PropertySymbol.currentTarget] = this;
        const browserSettings = window ? WindowBrowserSettingsReader_js_1.default.getSettings(window) : null;
        if (event.eventPhase !== EventPhaseEnum_js_1.default.capturing) {
            const onEventName = 'on' + event.type.toLowerCase();
            if (typeof this[onEventName] === 'function') {
                // We can end up in a never ending loop if the listener for the error event on Window also throws an error.
                if (window &&
                    (this !== window || event.type !== 'error') &&
                    !browserSettings?.disableErrorCapturing &&
                    browserSettings?.errorCapture === BrowserErrorCaptureEnum_js_1.default.tryAndCatch) {
                    WindowErrorUtility_js_1.default.captureError(window, this[onEventName].bind(this, event));
                }
                else {
                    this[onEventName].call(this, event);
                }
            }
        }
        if (this[PropertySymbol.listeners][event.type]) {
            // We need to clone the arrays because the listeners may remove themselves while we are iterating.
            const listeners = this[PropertySymbol.listeners][event.type].slice();
            const listenerOptions = this[PropertySymbol.listenerOptions][event.type].slice();
            for (let i = 0, max = listeners.length; i < max; i++) {
                const listener = listeners[i];
                const options = listenerOptions[i];
                if ((options?.capture && event.eventPhase !== EventPhaseEnum_js_1.default.capturing) ||
                    (!options?.capture && event.eventPhase === EventPhaseEnum_js_1.default.capturing)) {
                    continue;
                }
                if (options?.passive) {
                    event[PropertySymbol.isInPassiveEventListener] = true;
                }
                // We can end up in a never ending loop if the listener for the error event on Window also throws an error.
                if (window &&
                    (this !== window || event.type !== 'error') &&
                    !browserSettings?.disableErrorCapturing &&
                    browserSettings?.errorCapture === BrowserErrorCaptureEnum_js_1.default.tryAndCatch) {
                    if (listener.handleEvent) {
                        WindowErrorUtility_js_1.default.captureError(window, listener.handleEvent.bind(listener, event));
                    }
                    else {
                        WindowErrorUtility_js_1.default.captureError(window, listener.bind(this, event));
                    }
                }
                else {
                    if (listener.handleEvent) {
                        listener.handleEvent(event);
                    }
                    else {
                        listener.call(this, event);
                    }
                }
                event[PropertySymbol.isInPassiveEventListener] = false;
                if (options?.once) {
                    // At this time, listeners and listenersOptions are cloned arrays. When the original value is deleted,
                    // The value corresponding to the cloned array is not deleted. So we need to delete the value in the cloned array.
                    listeners.splice(i, 1);
                    listenerOptions.splice(i, 1);
                    this.removeEventListener(event.type, listener);
                    i--;
                    max--;
                }
                if (event[PropertySymbol.immediatePropagationStopped]) {
                    return !(event.cancelable && event.defaultPrevented);
                }
            }
        }
        return !(event.cancelable && event.defaultPrevented);
    }
    /**
     * Adds an event listener.
     *
     * TODO:
     * Was used by with IE8- and Opera. React believed Happy DOM was a legacy browser and used them, but that is no longer the case, so we should remove this method after that this is verified.
     *
     * @deprecated
     * @param type Event type.
     * @param listener Listener.
     */
    attachEvent(type, listener) {
        this.addEventListener(type.replace('on', ''), listener);
    }
    /**
     * Removes an event listener.
     *
     * TODO:
     * Was used by IE8- and Opera. React believed Happy DOM was a legacy browser and used them, but that is no longer the case, so we should remove this method after that this is verified.
     *
     * @deprecated
     * @param type Event type.
     * @param listener Listener.
     */
    detachEvent(type, listener) {
        this.removeEventListener(type.replace('on', ''), listener);
    }
}
_EventTarget_getWindow = function _EventTarget_getWindow() {
    if (this[PropertySymbol.ownerDocument]) {
        return this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow];
    }
    if (this[PropertySymbol.ownerWindow]) {
        return this[PropertySymbol.ownerWindow];
    }
    if (this.document) {
        return this;
    }
    return null;
};
exports.default = EventTarget;
//# sourceMappingURL=EventTarget.cjs.map