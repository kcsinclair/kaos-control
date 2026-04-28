import Attr from '../attr/Attr.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import HTMLElementNamedNodeMap from '../html-element/HTMLElementNamedNodeMap.js';
import HTMLButtonElement from './HTMLButtonElement.js';
/**
 * Named Node Map.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/NamedNodeMap
 */
export default class HTMLButtonElementNamedNodeMap extends HTMLElementNamedNodeMap {
    protected [PropertySymbol.ownerElement]: HTMLButtonElement;
    /**
     * @override
     */
    setNamedItem(item: Attr): Attr | null;
    /**
     * @override
     */
    [PropertySymbol.removeNamedItem](name: string): Attr | null;
}
//# sourceMappingURL=HTMLButtonElementNamedNodeMap.d.ts.map