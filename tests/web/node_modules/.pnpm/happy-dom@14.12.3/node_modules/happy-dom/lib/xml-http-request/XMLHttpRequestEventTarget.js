import EventTarget from '../event/EventTarget.js';
/**
 * References: https://xhr.spec.whatwg.org/#xmlhttprequesteventtarget.
 */
export default class XMLHttpRequestEventTarget extends EventTarget {
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
//# sourceMappingURL=XMLHttpRequestEventTarget.js.map