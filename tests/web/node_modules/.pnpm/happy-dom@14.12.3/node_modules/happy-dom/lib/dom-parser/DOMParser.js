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
var _DOMParser_instances, _DOMParser_window, _DOMParser_createDocument;
import * as PropertySymbol from '../PropertySymbol.js';
import XMLParser from '../xml-parser/XMLParser.js';
import DOMException from '../exception/DOMException.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
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
            throw new DOMException('Second parameter "mimeType" is mandatory.');
        }
        const newDocument = __classPrivateFieldGet(this, _DOMParser_instances, "m", _DOMParser_createDocument).call(this, mimeType);
        newDocument[PropertySymbol.childNodes].length = 0;
        newDocument[PropertySymbol.children].length = 0;
        const root = XMLParser.parse(newDocument, string, { evaluateScripts: true });
        let documentElement = null;
        let documentTypeNode = null;
        for (const node of root[PropertySymbol.childNodes]) {
            if (node['tagName'] === 'HTML') {
                documentElement = node;
            }
            else if (node[PropertySymbol.nodeType] === NodeTypeEnum.documentTypeNode) {
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
            throw new DOMException(`Unknown mime type "${mimeType}".`);
    }
};
export default DOMParser;
//# sourceMappingURL=DOMParser.js.map