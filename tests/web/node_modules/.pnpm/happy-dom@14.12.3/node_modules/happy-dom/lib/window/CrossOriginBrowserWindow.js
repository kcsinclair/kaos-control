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
var _CrossOriginBrowserWindow_targetWindow;
import EventTarget from '../event/EventTarget.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
/**
 * Browser window with limited access due to CORS restrictions in iframes.
 */
class CrossOriginBrowserWindow extends EventTarget {
    /**
     * Constructor.
     *
     * @param target Target window.
     * @param [parent] Parent window.
     */
    constructor(target, parent) {
        super();
        this.self = this;
        this.window = this;
        _CrossOriginBrowserWindow_targetWindow.set(this, void 0);
        this.parent = parent ?? this;
        this.top = parent ?? this;
        this.location = new Proxy({}, {
            get: () => {
                throw new DOMException(`Blocked a frame with origin "${this.parent.location.origin}" from accessing a cross-origin frame.`, DOMExceptionNameEnum.securityError);
            },
            set: () => {
                throw new DOMException(`Blocked a frame with origin "${this.parent.location.origin}" from accessing a cross-origin frame.`, DOMExceptionNameEnum.securityError);
            }
        });
        __classPrivateFieldSet(this, _CrossOriginBrowserWindow_targetWindow, target, "f");
    }
    /**
     * Returns the opener.
     *
     * @returns Opener.
     */
    get opener() {
        return __classPrivateFieldGet(this, _CrossOriginBrowserWindow_targetWindow, "f").opener;
    }
    /**
     * Returns the closed state.
     *
     * @returns Closed state.
     */
    get closed() {
        return __classPrivateFieldGet(this, _CrossOriginBrowserWindow_targetWindow, "f").closed;
    }
    /**
     * Shifts focus away from the window.
     */
    blur() {
        __classPrivateFieldGet(this, _CrossOriginBrowserWindow_targetWindow, "f").blur();
    }
    /**
     * Gives focus to the window.
     */
    focus() {
        __classPrivateFieldGet(this, _CrossOriginBrowserWindow_targetWindow, "f").focus();
    }
    /**
     * Closes the window.
     */
    close() {
        __classPrivateFieldGet(this, _CrossOriginBrowserWindow_targetWindow, "f").close();
    }
    /**
     * Safely enables cross-origin communication between Window objects; e.g., between a page and a pop-up that it spawned, or between a page and an iframe embedded within it.
     *
     * @param message Message.
     * @param [targetOrigin=*] Target origin.
     * @param transfer Transfer. Not implemented.
     */
    postMessage(message, targetOrigin = '*', transfer) {
        __classPrivateFieldGet(this, _CrossOriginBrowserWindow_targetWindow, "f").postMessage(message, targetOrigin, transfer);
    }
}
_CrossOriginBrowserWindow_targetWindow = new WeakMap();
export default CrossOriginBrowserWindow;
//# sourceMappingURL=CrossOriginBrowserWindow.js.map