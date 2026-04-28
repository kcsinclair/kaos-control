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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const EventTarget_js_1 = __importDefault(require("../event/EventTarget.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const Event_js_1 = __importDefault(require("../event/Event.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
/**
 * AbortSignal.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal
 */
class AbortSignal extends EventTarget_js_1.default {
    constructor() {
        super(...arguments);
        this.aborted = false;
        this.reason = null;
        this.onabort = null;
    }
    /**
     * Return a default description for the AbortSignal class.
     */
    get [Symbol.toStringTag]() {
        return 'AbortSignal';
    }
    /**
     * Aborts the signal.
     *
     * @param [reason] Reason.
     */
    [PropertySymbol.abort](reason) {
        if (this.aborted) {
            return;
        }
        this.reason =
            reason ||
                new DOMException_js_1.default('signal is aborted without reason', DOMExceptionNameEnum_js_1.default.abortError);
        this.aborted = true;
        this.dispatchEvent(new Event_js_1.default('abort'));
    }
    /**
     * Throws an "AbortError" if the signal has been aborted.
     */
    throwIfAborted() {
        if (this.aborted) {
            throw this.reason;
        }
    }
    /**
     * Returns an AbortSignal instance that has been set as aborted.
     *
     * @param [reason] Reason.
     * @returns AbortSignal instance.
     */
    static abort(reason) {
        const signal = new AbortSignal();
        signal.reason =
            reason ||
                new DOMException_js_1.default('signal is aborted without reason', DOMExceptionNameEnum_js_1.default.abortError);
        signal.aborted = true;
        return signal;
    }
}
exports.default = AbortSignal;
//# sourceMappingURL=AbortSignal.cjs.map