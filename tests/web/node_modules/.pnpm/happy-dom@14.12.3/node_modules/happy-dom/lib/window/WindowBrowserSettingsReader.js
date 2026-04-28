var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _a, _WindowBrowserSettingsReader_settings;
import * as PropertySymbol from '../PropertySymbol.js';
/**
 * Browser settings reader that will allow to read settings more securely as it is not possible to override a settings object to make DOM functionality act on it.
 */
class WindowBrowserSettingsReader {
    /**
     * Returns browser settings.
     *
     * @param window Window.
     * @returns Settings.
     */
    static getSettings(window) {
        const id = window[PropertySymbol.happyDOMSettingsID];
        if (id === undefined || !__classPrivateFieldGet(this, _a, "f", _WindowBrowserSettingsReader_settings)[id]) {
            return null;
        }
        return __classPrivateFieldGet(this, _a, "f", _WindowBrowserSettingsReader_settings)[id];
    }
    /**
     * Sets browser settings.
     *
     * @param window Window.
     * @param settings Settings.
     */
    static setSettings(window, settings) {
        if (window[PropertySymbol.happyDOMSettingsID] !== undefined) {
            return;
        }
        window[PropertySymbol.happyDOMSettingsID] = __classPrivateFieldGet(this, _a, "f", _WindowBrowserSettingsReader_settings).length;
        __classPrivateFieldGet(this, _a, "f", _WindowBrowserSettingsReader_settings).push(settings);
    }
    /**
     * Removes browser settings.
     *
     * @param window Window.
     */
    static removeSettings(window) {
        const id = window[PropertySymbol.happyDOMSettingsID];
        if (id !== undefined && __classPrivateFieldGet(this, _a, "f", _WindowBrowserSettingsReader_settings)[id]) {
            delete __classPrivateFieldGet(this, _a, "f", _WindowBrowserSettingsReader_settings)[id];
        }
        delete window[PropertySymbol.happyDOMSettingsID];
    }
}
_a = WindowBrowserSettingsReader;
_WindowBrowserSettingsReader_settings = { value: [] };
export default WindowBrowserSettingsReader;
//# sourceMappingURL=WindowBrowserSettingsReader.js.map