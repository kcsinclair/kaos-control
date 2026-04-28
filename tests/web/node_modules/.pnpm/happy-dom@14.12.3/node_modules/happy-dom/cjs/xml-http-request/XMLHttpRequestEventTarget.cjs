"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const EventTarget_js_1 = __importDefault(require("../event/EventTarget.cjs"));
/**
 * References: https://xhr.spec.whatwg.org/#xmlhttprequesteventtarget.
 */
class XMLHttpRequestEventTarget extends EventTarget_js_1.default {
    constructor() {
        super(...arguments);
        this.onloadstart = null;
        this.onprogress = null;
        this.onabort = null;
        this.onerror = null;
        this.onload = null;
        this.ontimeout = null;
        this.onloadend = null;
    }
}
exports.default = XMLHttpRequestEventTarget;
//# sourceMappingURL=XMLHttpRequestEventTarget.cjs.map