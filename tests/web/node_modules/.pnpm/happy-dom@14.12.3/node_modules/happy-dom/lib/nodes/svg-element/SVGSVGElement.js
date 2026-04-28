import SVGGraphicsElement from './SVGGraphicsElement.js';
import SVGRect from './SVGRect.js';
import SVGPoint from './SVGPoint.js';
import SVGLength from './SVGLength.js';
import SVGAngle from './SVGAngle.js';
import SVGNumber from './SVGNumber.js';
import SVGTransform from './SVGTransform.js';
import SVGAnimatedRect from './SVGAnimatedRect.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * SVGSVGElement.
 */
export default class SVGSVGElement extends SVGGraphicsElement {
    constructor() {
        super(...arguments);
        // Events
        this.onafterprint = null;
        this.onbeforeprint = null;
        this.onbeforeunload = null;
        this.ongamepadconnected = null;
        this.ongamepaddisconnected = null;
        this.onhashchange = null;
        this.onlanguagechange = null;
        this.onmessage = null;
        this.onmessageerror = null;
        this.onoffline = null;
        this.ononline = null;
        this.onpagehide = null;
        this.onpageshow = null;
        this.onpopstate = null;
        this.onrejectionhandled = null;
        this.onstorage = null;
        this.onunhandledrejection = null;
        this.onunload = null;
    }
    /**
     * Returns preserveAspectRatio.
     *
     * @returns PreserveAspectRatio.
     */
    get preserveAspectRatio() {
        return this.getAttributeNS(null, 'preserveAspectRatio') || 'xMidYMid meet';
    }
    /**
     * Sets preserveAspectRatio.
     *
     * @param preserveAspectRatio PreserveAspectRatio.
     */
    set preserveAspectRatio(preserveAspectRatio) {
        this.setAttributeNS(null, 'preserveAspectRatio', preserveAspectRatio);
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        return this.getAttributeNS(null, 'width') || '';
    }
    /**
     * Sets width.
     *
     * @param width Width.
     */
    set width(width) {
        this.setAttributeNS(null, 'width', width);
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        return this.getAttributeNS(null, 'height') || '';
    }
    /**
     * Sets height.
     *
     * @param height Height.
     */
    set height(height) {
        this.setAttributeNS(null, 'height', height);
    }
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        return this.getAttributeNS(null, 'x') || '';
    }
    /**
     * Sets x.
     *
     * @param x X.
     */
    set x(x) {
        this.setAttributeNS(null, 'x', x);
    }
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y() {
        return this.getAttributeNS(null, 'y') || '';
    }
    /**
     * Sets y.
     *
     * @param y Y.
     */
    set y(y) {
        this.setAttributeNS(null, 'y', y);
    }
    /**
     * Returns contentScriptType.
     *
     * @returns ContentScriptType.
     */
    get contentScriptType() {
        return this.getAttributeNS(null, 'contentScriptType') || '';
    }
    /**
     * Sets contentScriptType.
     *
     * @param contentScriptType ContentScriptType.
     */
    set contentScriptType(contentScriptType) {
        this.setAttributeNS(null, 'contentScriptType', contentScriptType);
    }
    /**
     * Returns currentScale.
     *
     * @returns CurrentScale.
     */
    get currentScale() {
        const currentScale = this.getAttributeNS(null, 'currentScale');
        if (currentScale !== null) {
            return parseFloat(currentScale);
        }
        return 1;
    }
    /**
     * Sets currentScale.
     *
     * @param currentScale CurrentScale.
     */
    set currentScale(currentScale) {
        this.setAttributeNS(null, 'currentScale', String(currentScale));
    }
    /**
     * Returns viewport.
     *
     * @returns SVG rect.
     */
    get viewport() {
        return new SVGRect();
    }
    /**
     * Returns current translate.
     *
     * @returns SVG point.
     */
    get currentTranslate() {
        return new SVGPoint();
    }
    /**
     * Returns view box.
     *
     * @returns Viewbox.
     */
    get viewBox() {
        const rect = new SVGAnimatedRect();
        const viewBox = this.getAttribute('viewBox');
        const list = viewBox.split(/\s+/);
        rect.baseVal.x = Number(list[0]);
        rect.baseVal.y = Number(list[1]);
        rect.baseVal.width = Number(list[2]);
        rect.baseVal.height = Number(list[3]);
        return rect;
    }
    /**
     * Pauses animation.
     */
    pauseAnimations() { }
    /**
     * Unpauses animation.
     */
    unpauseAnimations() { }
    /**
     * Returns "true" if animation is paused.
     *
     * @returns "true" if animation is paused.
     */
    animationsPaused() {
        return false;
    }
    /**
     * Returns the current time in seconds relative to the start time for the current SVG document fragment.
     *
     * @returns Current time.
     */
    getCurrentTime() {
        return 0;
    }
    /**
     * Sets current time.
     */
    setCurrentTime() { }
    /**
     * Returns intersection list.
     *
     * @returns Intersection list.
     */
    getIntersectionList() {
        return [];
    }
    /**
     * Returns enclousure list.
     *
     * @returns Enclousure list.
     */
    getEnclosureList() {
        return [];
    }
    /**
     * Returns true if the rendered content of the given element intersects the supplied rectangle.
     *
     * @returns Intersection state.
     */
    checkIntersection() {
        return false;
    }
    /**
     * Returns true if the rendered content of the given element is entirely contained within the supplied rectangle.
     *
     * @returns Enclousure state.
     */
    checkEnclosure() {
        return false;
    }
    /**
     * Unselects any selected objects, including any selections of text strings and type-in bars.
     */
    deselectAll() { }
    /**
     * Returns a number.
     *
     * @returns Number.
     */
    createSVGNumber() {
        return new SVGNumber();
    }
    /**
     * Returns a length.
     *
     * @returns Length.
     */
    createSVGLength() {
        return new SVGLength();
    }
    /**
     * Returns a angle.
     *
     * @returns Angle.
     */
    createSVGAngle() {
        return new SVGAngle();
    }
    /**
     * Returns a point.
     *
     * @returns Point.
     */
    createSVGPoint() {
        return new SVGPoint();
    }
    /**
     * Returns a rect.
     *
     * @returns Rect.
     */
    createSVGRect() {
        return new SVGRect();
    }
    /**
     * Returns a transform.
     *
     * @returns Transform.
     */
    createSVGTransform() {
        return new SVGTransform();
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
}
//# sourceMappingURL=SVGSVGElement.js.map