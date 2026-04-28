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
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const Window_js_1 = __importDefault(require("./Window.cjs"));
const buffer_1 = require("buffer");
/**
 * Browser window.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Window.
 */
class GlobalWindow extends Window_js_1.default {
    constructor() {
        super(...arguments);
        // Node.js Globals
        this.Array = globalThis.Array;
        this.ArrayBuffer = globalThis.ArrayBuffer;
        this.Boolean = globalThis.Boolean;
        this.Buffer = buffer_1.Buffer;
        this.DataView = globalThis.DataView;
        this.Date = globalThis.Date;
        this.Error = globalThis.Error;
        this.EvalError = globalThis.EvalError;
        this.Float32Array = globalThis.Float32Array;
        this.Float64Array = globalThis.Float64Array;
        this.Function = globalThis.Function;
        this.Infinity = globalThis.Infinity;
        this.Int16Array = globalThis.Int16Array;
        this.Int32Array = globalThis.Int32Array;
        this.Int8Array = globalThis.Int8Array;
        this.Intl = globalThis.Intl;
        this.JSON = globalThis.JSON;
        this.Map = globalThis.Map;
        this.Math = globalThis.Math;
        this.NaN = globalThis.NaN;
        this.Number = globalThis.Number;
        this.Object = globalThis.Object;
        this.Promise = globalThis.Promise;
        this.RangeError = globalThis.RangeError;
        this.ReferenceError = globalThis.ReferenceError;
        this.RegExp = globalThis.RegExp;
        this.Set = globalThis.Set;
        this.String = globalThis.String;
        this.Symbol = globalThis.Symbol;
        this.SyntaxError = globalThis.SyntaxError;
        this.TypeError = globalThis.TypeError;
        this.URIError = globalThis.URIError;
        this.Uint16Array = globalThis.Uint16Array;
        this.Uint32Array = globalThis.Uint32Array;
        this.Uint8Array = globalThis.Uint8Array;
        this.Uint8ClampedArray = globalThis.Uint8ClampedArray;
        this.WeakMap = globalThis.WeakMap;
        this.WeakSet = globalThis.WeakSet;
        this.decodeURI = globalThis.decodeURI;
        this.decodeURIComponent = globalThis.decodeURIComponent;
        this.encodeURI = globalThis.encodeURI;
        this.encodeURIComponent = globalThis.encodeURIComponent;
        this.eval = globalThis.eval;
        /**
         * @deprecated
         */
        this.escape = globalThis.escape;
        this.global = globalThis;
        this.isFinite = globalThis.isFinite;
        this.isNaN = globalThis.isNaN;
        this.parseFloat = globalThis.parseFloat;
        this.parseInt = globalThis.parseInt;
        this.undefined = globalThis.undefined;
        /**
         * @deprecated
         */
        this.unescape = globalThis.unescape;
        this.gc = globalThis.gc;
        this.v8debug = globalThis.v8debug;
    }
    /**
     * Setup of VM context.
     */
    [PropertySymbol.setupVMContext]() {
        // Do nothing
    }
}
exports.default = GlobalWindow;
//# sourceMappingURL=GlobalWindow.cjs.map