var __classPrivateFieldGet = (this && this.__classPrivateFieldGet) || function (receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
};
var _CookieContainer_cookies;
import CookieExpireUtility from './urilities/CookieExpireUtility.js';
import CookieURLUtility from './urilities/CookieURLUtility.js';
/**
 * Cookie Container.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cookie.
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie.
 */
class CookieContainer {
    constructor() {
        _CookieContainer_cookies.set(this, []);
    }
    /**
     * Adds cookies.
     *
     * @param cookies Cookies.
     */
    addCookies(cookies) {
        const indexMap = {};
        const getKey = (cookie) => `${cookie.key}-${cookie.originURL.hostname}-${cookie.path}-${typeof cookie.value}`;
        // Creates a map of cookie key, domain, path and value to index.
        for (let i = 0, max = __classPrivateFieldGet(this, _CookieContainer_cookies, "f").length; i < max; i++) {
            indexMap[getKey(__classPrivateFieldGet(this, _CookieContainer_cookies, "f")[i])] = i;
        }
        for (const cookie of cookies) {
            if (cookie?.key) {
                // Remove existing cookie with same name, domain and path.
                const index = indexMap[getKey(cookie)];
                if (index !== undefined) {
                    __classPrivateFieldGet(this, _CookieContainer_cookies, "f").splice(index, 1);
                }
                if (!CookieExpireUtility.hasExpired(cookie)) {
                    indexMap[getKey(cookie)] = __classPrivateFieldGet(this, _CookieContainer_cookies, "f").length;
                    __classPrivateFieldGet(this, _CookieContainer_cookies, "f").push(cookie);
                }
            }
        }
    }
    /**
     * Returns cookies.
     *
     * @param [url] URL.
     * @param [httpOnly] "true" if only http cookies should be returned.
     * @returns Cookies.
     */
    getCookies(url = null, httpOnly = false) {
        const cookies = [];
        for (const cookie of __classPrivateFieldGet(this, _CookieContainer_cookies, "f")) {
            if (!CookieExpireUtility.hasExpired(cookie) &&
                (!httpOnly || !cookie.httpOnly) &&
                (!url || CookieURLUtility.cookieMatchesURL(cookie, url || cookie.originURL))) {
                cookies.push(cookie);
            }
        }
        return cookies;
    }
}
_CookieContainer_cookies = new WeakMap();
export default CookieContainer;
//# sourceMappingURL=CookieContainer.js.map