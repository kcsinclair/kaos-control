var _a, _b, _c, _d, _e, _f, _g, _h, _j, _k, _l, _m, _o, _p, _q, _r, _s, _t;
import Node from '../node/Node.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMRect from './DOMRect.js';
import DOMTokenList from '../../dom-token-list/DOMTokenList.js';
import QuerySelector from '../../query-selector/QuerySelector.js';
import XMLParser from '../../xml-parser/XMLParser.js';
import XMLSerializer from '../../xml-serializer/XMLSerializer.js';
import ChildNodeUtility from '../child-node/ChildNodeUtility.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
import NonDocumentChildNodeUtility from '../child-node/NonDocumentChildNodeUtility.js';
import DOMException from '../../exception/DOMException.js';
import HTMLCollection from './HTMLCollection.js';
import DOMRectList from './DOMRectList.js';
import ElementUtility from './ElementUtility.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import ElementNamedNodeMap from './ElementNamedNodeMap.js';
import WindowErrorUtility from '../../window/WindowErrorUtility.js';
import WindowBrowserSettingsReader from '../../window/WindowBrowserSettingsReader.js';
import BrowserErrorCaptureEnum from '../../browser/enums/BrowserErrorCaptureEnum.js';
import NodeFactory from '../NodeFactory.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * Element.
 */
class Element extends Node {
    constructor() {
        super(...arguments);
        // Events
        this.oncancel = null;
        this.onerror = null;
        this.onscroll = null;
        this.onselect = null;
        this.onwheel = null;
        this.oncopy = null;
        this.oncut = null;
        this.onpaste = null;
        this.oncompositionend = null;
        this.oncompositionstart = null;
        this.oncompositionupdate = null;
        this.onblur = null;
        this.onfocus = null;
        this.onfocusin = null;
        this.onfocusout = null;
        this.onfullscreenchange = null;
        this.onfullscreenerror = null;
        this.onkeydown = null;
        this.onkeyup = null;
        this.onauxclick = null;
        this.onclick = null;
        this.oncontextmenu = null;
        this.ondblclick = null;
        this.onmousedown = null;
        this.onmouseenter = null;
        this.onmouseleave = null;
        this.onmousemove = null;
        this.onmouseout = null;
        this.onmouseover = null;
        this.onmouseup = null;
        this.ontouchcancel = null;
        this.ontouchend = null;
        this.ontouchmove = null;
        this.ontouchstart = null;
        // Internal properties
        this[_d] = new HTMLCollection();
        this[_e] = null;
        this[_f] = null;
        this[_g] = null;
        this[_h] = NodeTypeEnum.elementNode;
        this[_j] = this.constructor[PropertySymbol.tagName] || null;
        this[_k] = this.constructor[PropertySymbol.localName] || null;
        this[_l] = null;
        this[_m] = null;
        this[_o] = 0;
        this[_p] = 0;
        this[_q] = 0;
        this[_r] = 0;
        this[_s] = new ElementNamedNodeMap(this);
        this[_t] = this.constructor[PropertySymbol.namespaceURI] || null;
    }
    /**
     * Returns tag name.
     *
     * @returns Tag name.
     */
    get tagName() {
        return this[PropertySymbol.tagName];
    }
    /**
     * Returns prefix.
     *
     * @returns Prefix.
     */
    get prefix() {
        return this[PropertySymbol.prefix];
    }
    /**
     * Returns shadow root.
     *
     * @returns Shadow root.
     */
    get shadowRoot() {
        const shadowRoot = this[PropertySymbol.shadowRoot];
        return shadowRoot && shadowRoot[PropertySymbol.mode] === 'open' ? shadowRoot : null;
    }
    /**
     * Returns scroll height.
     *
     * @returns Scroll height.
     */
    get scrollHeight() {
        return this[PropertySymbol.scrollHeight];
    }
    /**
     * Returns scroll width.
     *
     * @returns Scroll width.
     */
    get scrollWidth() {
        return this[PropertySymbol.scrollWidth];
    }
    /**
     * Returns scroll top.
     *
     * @returns Scroll top.
     */
    get scrollTop() {
        return this[PropertySymbol.scrollTop];
    }
    /**
     * Sets scroll top.
     *
     * @param value Scroll top.
     */
    set scrollTop(value) {
        this[PropertySymbol.scrollTop] = value;
    }
    /**
     * Returns scroll left.
     *
     * @returns Scroll left.
     */
    get scrollLeft() {
        return this[PropertySymbol.scrollLeft];
    }
    /**
     * Sets scroll left.
     *
     * @param value Scroll left.
     */
    set scrollLeft(value) {
        this[PropertySymbol.scrollLeft] = value;
    }
    /**
     * Returns attributes.
     *
     * @returns Attributes.
     */
    get attributes() {
        return this[PropertySymbol.attributes];
    }
    /**
     * Returns namespace URI.
     *
     * @returns Namespace URI.
     */
    get namespaceURI() {
        return this[PropertySymbol.namespaceURI];
    }
    /**
     * Returns element children.
     */
    get children() {
        return this[PropertySymbol.children];
    }
    /**
     * Returns class list.
     *
     * @returns Class list.
     */
    get classList() {
        if (!this[PropertySymbol.classList]) {
            this[PropertySymbol.classList] = new DOMTokenList(this, 'class');
        }
        return this[PropertySymbol.classList];
    }
    /**
     * Returns ID.
     *
     * @returns ID.
     */
    get id() {
        return this.getAttribute('id') || '';
    }
    /**
     * Sets ID.
     *
     * @param id ID.
     */
    set id(id) {
        this.setAttribute('id', id);
    }
    /**
     * Returns class name.
     *
     * @returns Class name.
     */
    get className() {
        return this.getAttribute('class') || '';
    }
    /**
     * Sets class name.
     *
     * @param className Class name.
     */
    set className(className) {
        this.setAttribute('class', className);
    }
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName() {
        return this[PropertySymbol.tagName];
    }
    /**
     * Local name.
     *
     * @returns Local name.
     */
    get localName() {
        return this[PropertySymbol.localName];
    }
    /**
     * Returns role.
     *
     * @returns Role.
     */
    get role() {
        return this.getAttribute('role') || '';
    }
    /**
     * Sets role.
     *
     * @param role Role.
     */
    set role(role) {
        this.setAttribute('role', role);
    }
    /**
     * Previous element sibling.
     *
     * @returns Element.
     */
    get previousElementSibling() {
        return NonDocumentChildNodeUtility.previousElementSibling(this);
    }
    /**
     * Next element sibling.
     *
     * @returns Element.
     */
    get nextElementSibling() {
        return NonDocumentChildNodeUtility.nextElementSibling(this);
    }
    /**
     * Get text value of children.
     *
     * @returns Text content.
     */
    get textContent() {
        let result = '';
        for (const childNode of this[PropertySymbol.childNodes]) {
            if (childNode[PropertySymbol.nodeType] === NodeTypeEnum.elementNode ||
                childNode[PropertySymbol.nodeType] === NodeTypeEnum.textNode) {
                result += childNode.textContent;
            }
        }
        return result;
    }
    /**
     * Sets text content.
     *
     * @param textContent Text content.
     */
    set textContent(textContent) {
        for (const child of this[PropertySymbol.childNodes].slice()) {
            this.removeChild(child);
        }
        if (textContent) {
            this.appendChild(this[PropertySymbol.ownerDocument].createTextNode(textContent));
        }
    }
    /**
     * Returns inner HTML.
     *
     * @returns HTML.
     */
    get innerHTML() {
        return this.getInnerHTML();
    }
    /**
     * Sets inner HTML.
     *
     * @param html HTML.
     */
    set innerHTML(html) {
        for (const child of this[PropertySymbol.childNodes].slice()) {
            this.removeChild(child);
        }
        XMLParser.parse(this[PropertySymbol.ownerDocument], html, { rootNode: this });
    }
    /**
     * Returns outer HTML.
     *
     * @returns HTML.
     */
    get outerHTML() {
        return new XMLSerializer({ escapeEntities: false }).serializeToString(this);
    }
    /**
     * Returns outer HTML.
     *
     * @param html HTML.
     */
    set outerHTML(html) {
        this.replaceWith(html);
    }
    /**
     * First element child.
     *
     * @returns Element.
     */
    get firstElementChild() {
        return this[PropertySymbol.children][0] ?? null;
    }
    /**
     * Last element child.
     *
     * @returns Element.
     */
    get lastElementChild() {
        return this[PropertySymbol.children][this[PropertySymbol.children].length - 1] ?? null;
    }
    /**
     * Last element child.
     *
     * @returns Element.
     */
    get childElementCount() {
        return this[PropertySymbol.children].length;
    }
    /**
     * Returns slot.
     *
     * @returns Slot.
     */
    get slot() {
        return this.getAttributeNS(null, 'slot') || '';
    }
    /**
     * Returns slot.
     *
     * @param slot Slot.
     */
    set slot(title) {
        this.setAttribute('slot', title);
    }
    /**
     * Returns inner HTML and optionally the content of shadow roots.
     *
     * This is a feature implemented in Chromium, but not supported by Mozilla yet.
     *
     * @see https://web.dev/declarative-shadow-dom/
     * @see https://chromestatus.com/feature/5191745052606464
     * @param [options] Options.
     * @param [options.includeShadowRoots] Set to "true" to include shadow roots.
     * @returns HTML.
     */
    getInnerHTML(options) {
        const xmlSerializer = new XMLSerializer({
            includeShadowRoots: options && options.includeShadowRoots,
            escapeEntities: false
        });
        let xml = '';
        for (const node of this[PropertySymbol.childNodes]) {
            xml += xmlSerializer.serializeToString(node);
        }
        return xml;
    }
    /**
     * @override
     */
    [(PropertySymbol.observedAttributes, _a = PropertySymbol.tagName, _b = PropertySymbol.localName, _c = PropertySymbol.namespaceURI, _d = PropertySymbol.children, _e = PropertySymbol.classList, _f = PropertySymbol.isValue, _g = PropertySymbol.computedStyle, _h = PropertySymbol.nodeType, _j = PropertySymbol.tagName, _k = PropertySymbol.localName, _l = PropertySymbol.prefix, _m = PropertySymbol.shadowRoot, _o = PropertySymbol.scrollHeight, _p = PropertySymbol.scrollWidth, _q = PropertySymbol.scrollTop, _r = PropertySymbol.scrollLeft, _s = PropertySymbol.attributes, _t = PropertySymbol.namespaceURI, PropertySymbol.cloneNode)](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.tagName] = this[PropertySymbol.tagName];
        clone[PropertySymbol.localName] = this[PropertySymbol.localName];
        clone[PropertySymbol.namespaceURI] = this[PropertySymbol.namespaceURI];
        for (let i = 0, max = this[PropertySymbol.attributes].length; i < max; i++) {
            const attribute = this[PropertySymbol.attributes][i];
            clone[PropertySymbol.attributes].setNamedItem(Object.assign(this[PropertySymbol.ownerDocument].createAttributeNS(attribute[PropertySymbol.namespaceURI], attribute[PropertySymbol.name]), attribute));
        }
        if (deep) {
            for (const node of clone[PropertySymbol.childNodes]) {
                if (node[PropertySymbol.nodeType] === NodeTypeEnum.elementNode) {
                    clone[PropertySymbol.children].push(node);
                }
            }
        }
        return clone;
    }
    /**
     * @override
     */
    [PropertySymbol.appendChild](node) {
        // We do not call super here as this will be handled by ElementUtility to improve performance by avoiding validation and other checks.
        return ElementUtility.appendChild(this, node);
    }
    /**
     * @override
     */
    [PropertySymbol.removeChild](node) {
        // We do not call super here as this will be handled by ElementUtility to improve performance by avoiding validation and other checks.
        return ElementUtility.removeChild(this, node);
    }
    /**
     * @override
     */
    [PropertySymbol.insertBefore](newNode, referenceNode) {
        // We do not call super here as this will be handled by ElementUtility to improve performance by avoiding validation and other checks.
        return ElementUtility.insertBefore(this, newNode, referenceNode);
    }
    /**
     * Removes the node from its parent.
     */
    remove() {
        ChildNodeUtility.remove(this);
    }
    /**
     * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceWith(...nodes) {
        ChildNodeUtility.replaceWith(this, ...nodes);
    }
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    before(...nodes) {
        ChildNodeUtility.before(this, ...nodes);
    }
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    after(...nodes) {
        ChildNodeUtility.after(this, ...nodes);
    }
    /**
     * Inserts a set of Node objects or DOMString objects after the last child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    append(...nodes) {
        ParentNodeUtility.append(this, ...nodes);
    }
    /**
     * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    prepend(...nodes) {
        ParentNodeUtility.prepend(this, ...nodes);
    }
    /**
     * Replaces the existing children of a node with a specified new set of children.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceChildren(...nodes) {
        ParentNodeUtility.replaceChildren(this, ...nodes);
    }
    /**
     * Inserts a node to the given position.
     *
     * @param position Position to insert element.
     * @param element Node to insert.
     * @returns Inserted node or null if couldn't insert.
     */
    insertAdjacentElement(position, element) {
        if (position === 'beforebegin') {
            if (!this.parentElement) {
                return null;
            }
            this.parentElement.insertBefore(element, this);
        }
        else if (position === 'afterbegin') {
            this.insertBefore(element, this.firstChild);
        }
        else if (position === 'beforeend') {
            this.appendChild(element);
        }
        else if (position === 'afterend') {
            if (!this.parentElement) {
                return null;
            }
            this.parentElement.insertBefore(element, this.nextSibling);
        }
        return element;
    }
    /**
     * Inserts an HTML string to the given position.
     *
     * @param position Position to insert text.
     * @param text HTML string to insert.
     */
    insertAdjacentHTML(position, text) {
        for (const node of (XMLParser.parse(this[PropertySymbol.ownerDocument], text))[PropertySymbol.childNodes].slice()) {
            this.insertAdjacentElement(position, node);
        }
    }
    /**
     * Inserts text to the given position.
     *
     * @param position Position to insert text.
     * @param text String to insert.
     */
    insertAdjacentText(position, text) {
        if (!text) {
            return;
        }
        const textNode = this[PropertySymbol.ownerDocument].createTextNode(text);
        this.insertAdjacentElement(position, textNode);
    }
    /**
     * Sets an attribute.
     *
     * @param name Name.
     * @param value Value.
     */
    setAttribute(name, value) {
        const attribute = this[PropertySymbol.ownerDocument].createAttributeNS(null, name);
        attribute[PropertySymbol.value] = String(value);
        this.setAttributeNode(attribute);
    }
    /**
     * Sets a namespace attribute.
     *
     * @param namespaceURI Namespace URI.
     * @param name Name.
     * @param value Value.
     */
    setAttributeNS(namespaceURI, name, value) {
        const attribute = this[PropertySymbol.ownerDocument].createAttributeNS(namespaceURI, name);
        attribute[PropertySymbol.value] = String(value);
        this.setAttributeNode(attribute);
    }
    /**
     * Returns attribute names.
     *
     * @returns Attribute names.
     */
    getAttributeNames() {
        const attributeNames = [];
        for (let i = 0, max = this[PropertySymbol.attributes].length; i < max; i++) {
            attributeNames.push(this[PropertySymbol.attributes][i][PropertySymbol.name]);
        }
        return attributeNames;
    }
    /**
     * Returns attribute value.
     *
     * @param name Name.
     */
    getAttribute(name) {
        const attribute = this.getAttributeNode(name);
        if (attribute) {
            return attribute[PropertySymbol.value];
        }
        return null;
    }
    /**
     * Toggle an attribute.
     * Returns `true` if attribute name is eventually present, and `false` otherwise.
     *
     * @param name A DOMString specifying the name of the attribute to be toggled.
     * @param force A boolean value to determine whether the attribute should be added or removed, no matter whether the attribute is present or not at the moment.
     */
    toggleAttribute(name, force) {
        name = name.toLowerCase();
        const attribute = this.getAttributeNode(name);
        if (attribute) {
            if (force === true) {
                return true;
            }
            this.removeAttributeNode(attribute);
            return false;
        }
        if (force === false) {
            return false;
        }
        this.setAttribute(name, '');
        return true;
    }
    /**
     * Returns namespace attribute value.
     *
     * @param namespace Namespace URI.
     * @param localName Local name.
     */
    getAttributeNS(namespace, localName) {
        const attribute = this.getAttributeNodeNS(namespace, localName);
        if (attribute) {
            return attribute[PropertySymbol.value];
        }
        return null;
    }
    /**
     * Returns a boolean value indicating whether the specified element has the attribute or not.
     *
     * @param name Attribute name.
     * @returns True if attribute exists, false otherwise.
     */
    hasAttribute(name) {
        return !!this.getAttributeNode(name);
    }
    /**
     * Returns a boolean value indicating whether the specified element has the namespace attribute or not.
     *
     * @param namespace Namespace URI.
     * @param localName Local name.
     * @returns True if attribute exists, false otherwise.
     */
    hasAttributeNS(namespace, localName) {
        return this[PropertySymbol.attributes].getNamedItemNS(namespace, localName) !== null;
    }
    /**
     * Returns "true" if the element has attributes.
     *
     * @returns "true" if the element has attributes.
     */
    hasAttributes() {
        return this[PropertySymbol.attributes].length > 0;
    }
    /**
     * Removes an attribute.
     *
     * @param name Name.
     */
    removeAttribute(name) {
        try {
            this[PropertySymbol.attributes].removeNamedItem(name);
        }
        catch (error) {
            // Ignore DOMException when the attribute does not exist.
        }
    }
    /**
     * Removes a namespace attribute.
     *
     * @param namespace Namespace URI.
     * @param localName Local name.
     */
    removeAttributeNS(namespace, localName) {
        this[PropertySymbol.attributes].removeNamedItemNS(namespace, localName);
    }
    /**
     * Attaches a shadow root.
     *
     * @param init Shadow root init.
     * @param init.mode Shadow root mode.
     * @returns Shadow root.
     */
    attachShadow(init) {
        if (this[PropertySymbol.shadowRoot]) {
            throw new DOMException('Shadow root has already been attached.');
        }
        const shadowRoot = NodeFactory.createNode(this[PropertySymbol.ownerDocument], this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].ShadowRoot);
        this[PropertySymbol.shadowRoot] = shadowRoot;
        shadowRoot[PropertySymbol.host] = this;
        shadowRoot[PropertySymbol.mode] = init.mode;
        shadowRoot[PropertySymbol.connectToNode](this);
        return this[PropertySymbol.shadowRoot];
    }
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString() {
        return this.outerHTML;
    }
    /**
     * Returns the size of an element and its position relative to the viewport.
     *
     * @returns DOM rect.
     */
    getBoundingClientRect() {
        // TODO: Not full implementation
        return new DOMRect();
    }
    /**
     * Returns a collection of DOMRect objects that indicate the bounding rectangles for each CSS border box in a client.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/Element/getClientRects
     * @returns DOM rect list.
     */
    getClientRects() {
        // TODO: Not full implementation
        const domRectList = new DOMRectList();
        domRectList.push(this.getBoundingClientRect());
        return domRectList;
    }
    /**
     * The matches() method checks to see if the Element would be selected by the provided selectorString.
     *
     * @param selector Selector.
     * @returns "true" if matching.
     */
    matches(selector) {
        return !!QuerySelector.matches(this, selector);
    }
    /**
     * Traverses the Element and its parents (heading toward the document root) until it finds a node that matches the provided selector string.
     *
     * @param selector Selector.
     * @returns Closest matching element.
     */
    closest(selector) {
        // eslint-disable-next-line
        let parent = this;
        while (parent) {
            if (QuerySelector.matches(parent, selector)) {
                return parent;
            }
            parent = parent.parentElement;
        }
        return null;
    }
    /**
     * Query CSS selector to find matching elments.
     *
     * @param selector CSS selector.
     * @returns Matching elements.
     */
    querySelectorAll(selector) {
        return QuerySelector.querySelectorAll(this, selector);
    }
    /**
     * Query CSS Selector to find matching node.
     *
     * @param selector CSS selector.
     * @returns Matching element.
     */
    querySelector(selector) {
        return QuerySelector.querySelector(this, selector);
    }
    /**
     * Returns an elements by class name.
     *
     * @param className Tag name.
     * @returns Matching element.
     */
    getElementsByClassName(className) {
        return ParentNodeUtility.getElementsByClassName(this, className);
    }
    /**
     * Returns an elements by tag name.
     *
     * @param tagName Tag name.
     * @returns Matching element.
     */
    getElementsByTagName(tagName) {
        return ParentNodeUtility.getElementsByTagName(this, tagName);
    }
    /**
     * Returns an elements by tag name and namespace.
     *
     * @param namespaceURI Namespace URI.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    getElementsByTagNameNS(namespaceURI, tagName) {
        return ParentNodeUtility.getElementsByTagNameNS(this, namespaceURI, tagName);
    }
    /**
     * The setAttributeNode() method adds a new Attr node to the specified element.
     *
     * @param attribute Attribute.
     * @returns Replaced attribute.
     */
    setAttributeNode(attribute) {
        return this[PropertySymbol.attributes].setNamedItem(attribute);
    }
    /**
     * The setAttributeNodeNS() method adds a new Attr node to the specified element.
     *
     * @param attribute Attribute.
     * @returns Replaced attribute.
     */
    setAttributeNodeNS(attribute) {
        return this[PropertySymbol.attributes].setNamedItemNS(attribute);
    }
    /**
     * Returns an Attr node.
     *
     * @param name Name.
     * @returns Replaced attribute.
     */
    getAttributeNode(name) {
        return this[PropertySymbol.attributes].getNamedItem(name);
    }
    /**
     * Returns a namespaced Attr node.
     *
     * @param namespace Namespace.
     * @param localName Name.
     * @returns Replaced attribute.
     */
    getAttributeNodeNS(namespace, localName) {
        return this[PropertySymbol.attributes].getNamedItemNS(namespace, localName);
    }
    /**
     * Removes an Attr node.
     *
     * @param attribute Attribute.
     * @returns Removed attribute.
     */
    removeAttributeNode(attribute) {
        return this[PropertySymbol.attributes].removeNamedItem(attribute[PropertySymbol.name]);
    }
    /**
     * Removes an Attr node.
     *
     * @param attribute Attribute.
     * @returns Removed attribute.
     */
    removeAttributeNodeNS(attribute) {
        return this[PropertySymbol.attributes].removeNamedItemNS(attribute[PropertySymbol.namespaceURI], attribute.localName);
    }
    /**
     * Scrolls to a particular set of coordinates.
     *
     * @param x X position or options object.
     * @param y Y position.
     */
    scroll(x, y) {
        if (typeof x === 'object') {
            if (x.behavior === 'smooth') {
                this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].setTimeout(() => {
                    if (x.top !== undefined) {
                        this.scrollTop = x.top;
                    }
                    if (x.left !== undefined) {
                        this.scrollLeft = x.left;
                    }
                });
            }
            else {
                if (x.top !== undefined) {
                    this.scrollTop = x.top;
                }
                if (x.left !== undefined) {
                    this.scrollLeft = x.left;
                }
            }
        }
        else if (x !== undefined && y !== undefined) {
            this.scrollLeft = x;
            this.scrollTop = y;
        }
    }
    /**
     * Scrolls to a particular set of coordinates.
     *
     * @param x X position or options object.
     * @param y Y position.
     */
    scrollTo(x, y) {
        this.scroll(x, y);
    }
    /**
     * Scrolls the element's ancestor containers such that the element on which scrollIntoView() is called is visible to the user.
     *
     * @param [_options] Options.
     */
    scrollIntoView(_options) {
        // Do nothing
    }
    /**
     * @override
     */
    dispatchEvent(event) {
        const returnValue = super.dispatchEvent(event);
        const browserSettings = WindowBrowserSettingsReader.getSettings(this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow]);
        if (browserSettings &&
            !browserSettings.disableJavaScriptEvaluation &&
            (event.eventPhase === EventPhaseEnum.atTarget ||
                event.eventPhase === EventPhaseEnum.bubbling) &&
            !event[PropertySymbol.immediatePropagationStopped]) {
            const attribute = this.getAttribute('on' + event.type);
            if (attribute && !event[PropertySymbol.immediatePropagationStopped]) {
                const code = `//# sourceURL=${this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].location.href}\n${attribute}`;
                if (browserSettings.disableErrorCapturing ||
                    browserSettings.errorCapture !== BrowserErrorCaptureEnum.tryAndCatch) {
                    this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].eval(code);
                }
                else {
                    WindowErrorUtility.captureError(this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow], () => this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].eval(code));
                }
            }
        }
        return returnValue;
    }
}
Element[_a] = null;
Element[_b] = null;
Element[_c] = null;
export default Element;
//# sourceMappingURL=Element.js.map