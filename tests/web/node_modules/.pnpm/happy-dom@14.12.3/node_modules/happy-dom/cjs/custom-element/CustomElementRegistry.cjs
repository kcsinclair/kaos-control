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
var _CustomElementRegistry_instances, _CustomElementRegistry_window, _CustomElementRegistry_isValidCustomElementName, _a, _b, _c;
Object.defineProperty(exports, "__esModule", { value: true });
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const NamespaceURI_js_1 = __importDefault(require("../config/NamespaceURI.cjs"));
/**
 * Custom elements registry.
 */
class CustomElementRegistry {
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window) {
        _CustomElementRegistry_instances.add(this);
        this[_a] = {};
        this[_b] = new Map();
        this[_c] = {};
        _CustomElementRegistry_window.set(this, void 0);
        __classPrivateFieldSet(this, _CustomElementRegistry_window, window, "f");
    }
    /**
     * Defines a custom element class.
     *
     * @param name Tag name of element.
     * @param elementClass Element class.
     * @param [options] Options.
     * @param [options.extends] Extends tag name.
     */
    define(name, elementClass, options) {
        if (!__classPrivateFieldGet(this, _CustomElementRegistry_instances, "m", _CustomElementRegistry_isValidCustomElementName).call(this, name)) {
            throw new DOMException_js_1.default(`Failed to execute 'define' on 'CustomElementRegistry': "${name}" is not a valid custom element name`);
        }
        if (this[PropertySymbol.registry][name]) {
            throw new DOMException_js_1.default(`Failed to execute 'define' on 'CustomElementRegistry': the name "${name}" has already been used with this registry`);
        }
        if (this[PropertySymbol.registedClass].has(elementClass)) {
            throw new DOMException_js_1.default("Failed to execute 'define' on 'CustomElementRegistry': this constructor has already been used with this registry");
        }
        const tagName = name.toUpperCase();
        elementClass[PropertySymbol.ownerDocument] = __classPrivateFieldGet(this, _CustomElementRegistry_window, "f").document;
        elementClass[PropertySymbol.tagName] = tagName;
        elementClass[PropertySymbol.localName] = name;
        elementClass[PropertySymbol.namespaceURI] = NamespaceURI_js_1.default.html;
        this[PropertySymbol.registry][name] = {
            elementClass,
            extends: options && options.extends ? options.extends.toLowerCase() : null
        };
        this[PropertySymbol.registedClass].set(elementClass, name);
        // ObservedAttributes should only be called once by CustomElementRegistry (see #117)
        if (elementClass.prototype.attributeChangedCallback) {
            elementClass[PropertySymbol.observedAttributes] = elementClass.observedAttributes;
        }
        if (this[PropertySymbol.callbacks][name]) {
            const callbacks = this[PropertySymbol.callbacks][name];
            delete this[PropertySymbol.callbacks][name];
            for (const callback of callbacks) {
                callback();
            }
        }
    }
    /**
     * Returns a defined element class.
     *
     * @param name Tag name of element.
     * @returns HTMLElement Class defined or undefined.
     */
    get(name) {
        return this[PropertySymbol.registry][name]?.elementClass;
    }
    /**
     * Upgrades a custom element directly, even before it is connected to its shadow root.
     *
     * Not implemented yet.
     *
     * @param _root Root node.
     */
    upgrade(_root) {
        // Do nothing
    }
    /**
     * When defined.
     *
     * @param name Tag name of element.
     */
    whenDefined(name) {
        if (!__classPrivateFieldGet(this, _CustomElementRegistry_instances, "m", _CustomElementRegistry_isValidCustomElementName).call(this, name)) {
            return Promise.reject(new DOMException_js_1.default(`Invalid custom element name: "${name}"`));
        }
        if (this.get(name)) {
            return Promise.resolve();
        }
        return new Promise((resolve) => {
            this[PropertySymbol.callbacks][name] = this[PropertySymbol.callbacks][name] || [];
            this[PropertySymbol.callbacks][name].push(resolve);
        });
    }
    /**
     * Reverse lookup searching for name by given element class.
     *
     * @param elementClass Class constructor.
     * @returns Found tag name or `null`.
     */
    getName(elementClass) {
        return this[PropertySymbol.registedClass].get(elementClass) || null;
    }
    /**
     * Destroys the registry.
     */
    [(_CustomElementRegistry_window = new WeakMap(), _CustomElementRegistry_instances = new WeakSet(), _a = PropertySymbol.registry, _b = PropertySymbol.registedClass, _c = PropertySymbol.callbacks, PropertySymbol.destroy)]() {
        for (const entity of Object.values(this[PropertySymbol.registry])) {
            entity.elementClass[PropertySymbol.ownerDocument] = null;
        }
        this[PropertySymbol.registry] = {};
        this[PropertySymbol.registedClass] = new Map();
        this[PropertySymbol.callbacks] = {};
    }
}
_CustomElementRegistry_isValidCustomElementName = function _CustomElementRegistry_isValidCustomElementName(name) {
    // Validation criteria based on:
    // https://html.spec.whatwg.org/multipage/custom-elements.html#valid-custom-element-name
    const PCENChar = '[-_.]|[0-9]|[a-z]|\u{B7}|[\u{C0}-\u{D6}]|[\u{D8}-\u{F6}]' +
        '|[\u{F8}-\u{37D}]|[\u{37F}-\u{1FFF}]' +
        '|[\u{200C}-\u{200D}]|[\u{203F}-\u{2040}]|[\u{2070}-\u{218F}]' +
        '|[\u{2C00}-\u{2FEF}]|[\u{3001}-\u{D7FF}]' +
        '|[\u{F900}-\u{FDCF}]|[\u{FDF0}-\u{FFFD}]|[\u{10000}-\u{EFFFF}]';
    const PCEN = new RegExp(`^[a-z](${PCENChar})*-(${PCENChar})*$`, 'u');
    const reservedNames = [
        'annotation-xml',
        'color-profile',
        'font-face',
        'font-face-src',
        'font-face-uri',
        'font-face-format',
        'font-face-name',
        'missing-glyph'
    ];
    return PCEN.test(name) && !reservedNames.includes(name);
};
exports.default = CustomElementRegistry;
//# sourceMappingURL=CustomElementRegistry.cjs.map