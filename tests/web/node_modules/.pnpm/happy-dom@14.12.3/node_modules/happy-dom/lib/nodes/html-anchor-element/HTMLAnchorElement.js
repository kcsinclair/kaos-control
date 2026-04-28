var _a, _b;
import HTMLElement from '../html-element/HTMLElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMTokenList from '../../dom-token-list/DOMTokenList.js';
import URL from '../../url/URL.js';
import HTMLAnchorElementNamedNodeMap from './HTMLAnchorElementNamedNodeMap.js';
import EventPhaseEnum from '../../event/EventPhaseEnum.js';
import MouseEvent from '../../event/events/MouseEvent.js';
/**
 * HTML Anchor Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLAnchorElement.
 */
class HTMLAnchorElement extends HTMLElement {
    constructor() {
        super(...arguments);
        this[_a] = new HTMLAnchorElementNamedNodeMap(this);
        this[_b] = null;
    }
    /**
     * Returns download.
     *
     * @returns download.
     */
    get download() {
        return this.getAttribute('download') || '';
    }
    /**
     * Sets download.
     *
     * @param download Download.
     */
    set download(download) {
        this.setAttribute('download', download);
    }
    /**
     * Returns hash.
     *
     * @returns Hash.
     */
    get hash() {
        const href = this.getAttribute('href');
        if (href.startsWith('#')) {
            return href;
        }
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return '';
        }
        return url.hash;
    }
    /**
     * Sets hash.
     *
     * @param hash Hash.
     */
    set hash(hash) {
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return;
        }
        url.hash = hash;
        this.href = url.href;
    }
    /**
     * Returns href.
     *
     * @returns Href.
     */
    get href() {
        if (!this.hasAttribute('href')) {
            return '';
        }
        try {
            return new URL(this.getAttribute('href'), this[PropertySymbol.ownerDocument].location.href)
                .href;
        }
        catch (e) {
            return this.getAttribute('href');
        }
    }
    /**
     * Sets href.
     *
     * @param href Href.
     */
    set href(href) {
        this.setAttribute('href', href);
    }
    /**
     * Returns hreflang.
     *
     * @returns Hreflang.
     */
    get hreflang() {
        return this.getAttribute('hreflang') || '';
    }
    /**
     * Sets hreflang.
     *
     * @param hreflang Hreflang.
     */
    set hreflang(hreflang) {
        this.setAttribute('hreflang', hreflang);
    }
    /**
     * Returns the hyperlink's URL's origin.
     *
     * @returns Origin.
     */
    get origin() {
        try {
            return new URL(this.href).origin;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Returns ping.
     *
     * @returns Ping.
     */
    get ping() {
        return this.getAttribute('ping') || '';
    }
    /**
     * Sets ping.
     *
     * @param ping Ping.
     */
    set ping(ping) {
        this.setAttribute('ping', ping);
    }
    /**
     * Returns protocol.
     *
     * @returns Protocol.
     */
    get protocol() {
        try {
            return new URL(this.href).protocol;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets protocol.
     *
     * @param protocol Protocol.
     */
    set protocol(protocol) {
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return;
        }
        url.protocol = protocol;
        this.href = url.href;
    }
    /**
     * Returns username.
     *
     * @returns Username.
     */
    get username() {
        try {
            return new URL(this.href).username;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets username.
     *
     * @param username Username.
     */
    set username(username) {
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return;
        }
        url.username = username;
        this.href = url.href;
    }
    /**
     * Returns password.
     *
     * @returns Password.
     */
    get password() {
        try {
            return new URL(this.href).password;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets password.
     *
     * @param password Password.
     */
    set password(password) {
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return;
        }
        url.password = password;
        this.href = url.href;
    }
    /**
     * Returns pathname.
     *
     * @returns Pathname.
     */
    get pathname() {
        try {
            return new URL(this.href).pathname;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets pathname.
     *
     * @param pathname Pathname.
     */
    set pathname(pathname) {
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return;
        }
        url.pathname = pathname;
        this.href = url.href;
    }
    /**
     * Returns port.
     *
     * @returns Port.
     */
    get port() {
        try {
            return new URL(this.href).port;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets port.
     *
     * @param port Port.
     */
    set port(port) {
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return;
        }
        url.port = port;
        this.href = url.href;
    }
    /**
     * Returns host.
     *
     * @returns Host.
     */
    get host() {
        try {
            return new URL(this.href).host;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets host.
     *
     * @param host Host.
     */
    set host(host) {
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return;
        }
        url.host = host;
        this.href = url.href;
    }
    /**
     * Returns hostname.
     *
     * @returns Hostname.
     */
    get hostname() {
        try {
            return new URL(this.href).hostname;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets hostname.
     *
     * @param hostname Hostname.
     */
    set hostname(hostname) {
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return;
        }
        url.hostname = hostname;
        this.href = url.href;
    }
    /**
     * Returns referrerPolicy.
     *
     * @returns Referrer Policy.
     */
    get referrerPolicy() {
        return this.getAttribute('referrerPolicy') || '';
    }
    /**
     * Sets referrerPolicy.
     *
     * @param referrerPolicy Referrer Policy.
     */
    set referrerPolicy(referrerPolicy) {
        this.setAttribute('referrerPolicy', referrerPolicy);
    }
    /**
     * Returns rel.
     *
     * @returns Rel.
     */
    get rel() {
        return this.getAttribute('rel') || '';
    }
    /**
     * Sets rel.
     *
     * @param rel Rel.
     */
    set rel(rel) {
        this.setAttribute('rel', rel);
    }
    /**
     * Returns rel list.
     *
     * @returns Rel list.
     */
    get relList() {
        if (!this[PropertySymbol.relList]) {
            this[PropertySymbol.relList] = new DOMTokenList(this, 'rel');
        }
        return this[PropertySymbol.relList];
    }
    /**
     * Returns search.
     *
     * @returns Search.
     */
    get search() {
        try {
            return new URL(this.href).search;
        }
        catch (e) {
            return '';
        }
    }
    /**
     * Sets search.
     *
     * @param search Search.
     */
    set search(search) {
        let url;
        try {
            url = new URL(this.href);
        }
        catch (e) {
            return;
        }
        url.search = search;
        this.href = url.href;
    }
    /**
     * Returns target.
     *
     * @returns target.
     */
    get target() {
        return this.getAttribute('target') || '';
    }
    /**
     * Sets target.
     *
     * @param target Target.
     */
    set target(target) {
        this.setAttribute('target', target);
    }
    /**
     * Returns text.
     *
     * @returns text.
     */
    get text() {
        return this.textContent;
    }
    /**
     * Sets text.
     *
     * @param text Text.
     */
    set text(text) {
        this.textContent = text;
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || '';
    }
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type);
    }
    /**
     * @override
     */
    toString() {
        return this.href;
    }
    /**
     * @override
     */
    dispatchEvent(event) {
        const returnValue = super.dispatchEvent(event);
        if (event.type === 'click' &&
            event instanceof MouseEvent &&
            event.eventPhase === EventPhaseEnum.none &&
            !event.defaultPrevented) {
            const href = this.href;
            if (href) {
                this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].open(href, this.target || '_self');
                if (this[PropertySymbol.ownerDocument][PropertySymbol.ownerWindow].closed) {
                    event.stopImmediatePropagation();
                }
            }
        }
        return returnValue;
    }
}
_a = PropertySymbol.attributes, _b = PropertySymbol.relList;
export default HTMLAnchorElement;
//# sourceMappingURL=HTMLAnchorElement.js.map