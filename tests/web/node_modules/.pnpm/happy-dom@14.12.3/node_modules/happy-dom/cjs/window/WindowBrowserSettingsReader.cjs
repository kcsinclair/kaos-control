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
var _a, _WindowBrowserSettingsReader_settings;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
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
exports.default = WindowBrowserSettingsReader;
//# sourceMappingURL=WindowBrowserSettingsReader.cjs.map