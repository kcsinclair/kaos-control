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
var _DOMParser_instances, _DOMParser_window, _DOMParser_createDocument;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const XMLParser_js_1 = __importDefault(require("../xml-parser/XMLParser.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../nodes/node/NodeTypeEnum.cjs"));
/**
 * DOM parser.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DOMParser.
 */
class DOMParser {
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window) {
        _DOMParser_instances.add(this);
        _DOMParser_window.set(this, void 0);
        __classPrivateFieldSet(this, _DOMParser_window, window, "f");
    }
    /**
     * Parses HTML and returns a root element.
     *
     * @param string HTML data.
     * @param mimeType Mime type.
     * @returns Root element.
     */
    parseFromString(string, mimeType) {
        if (!mimeType) {
            throw new DOMException_js_1.default('Second parameter "mimeType" is mandatory.');
        }
        const newDocument = __classPrivateFieldGet(this, _DOMParser_instances, "m", _DOMParser_createDocument).call(this, mimeType);
        newDocument[PropertySymbol.childNodes].length = 0;
        newDocument[PropertySymbol.children].length = 0;
        const root = XMLParser_js_1.default.parse(newDocument, string, { evaluateScripts: true });
        let documentElement = null;
        let documentTypeNode = null;
        for (const node of root[PropertySymbol.childNodes]) {
            if (node['tagName'] === 'HTML') {
                documentElement = node;
            }
            else if (node[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.documentTypeNode) {
                documentTypeNode = node;
            }
            if (documentElement && documentTypeNode) {
                break;
            }
        }
        if (documentElement) {
            if (documentTypeNode) {
                newDocument.appendChild(documentTypeNode);
            }
            newDocument.appendChild(documentElement);
            const body = newDocument.body;
            if (body) {
                for (const child of root[PropertySymbol.childNodes].slice()) {
                    body.appendChild(child);
                }
            }
        }
        else {
            switch (mimeType) {
                case 'image/svg+xml':
                    {
                        for (const node of root[PropertySymbol.childNodes].slice()) {
                            newDocument.appendChild(node);
                        }
                    }
                    break;
                case 'text/html':
                default:
                    {
                        const documentElement = newDocument.createElement('html');
                        const bodyElement = newDocument.createElement('body');
                        const headElement = newDocument.createElement('head');
                        documentElement.appendChild(headElement);
                        documentElement.appendChild(bodyElement);
                        newDocument.appendChild(documentElement);
                        for (const node of root[PropertySymbol.childNodes].slice()) {
                            bodyElement.appendChild(node);
                        }
                    }
                    break;
            }
        }
        return newDocument;
    }
}
_DOMParser_window = new WeakMap(), _DOMParser_instances = new WeakSet(), _DOMParser_createDocument = function _DOMParser_createDocument(mimeType) {
    switch (mimeType) {
        case 'text/html':
            return new (__classPrivateFieldGet(this, _DOMParser_window, "f").HTMLDocument)();
        case 'image/svg+xml':
            return new (__classPrivateFieldGet(this, _DOMParser_window, "f").SVGDocument)();
        case 'text/xml':
        case 'application/xml':
        case 'application/xhtml+xml':
            return new (__classPrivateFieldGet(this, _DOMParser_window, "f").XMLDocument)();
        default:
            throw new DOMException_js_1.default(`Unknown mime type "${mimeType}".`);
    }
};
exports.default = DOMParser;
//# sourceMappingURL=DOMParser.cjs.map