var _a, _b, _c, _d, _e;
import * as PropertySymbol from '../PropertySymbol.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import EventPhaseEnum from './EventPhaseEnum.js';
/**
 * Event.
 */
class Event {
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        this.defaultPrevented = false;
        this.eventPhase = EventPhaseEnum.none;
        this.timeStamp = performance.now();
        this.NONE = EventPhaseEnum.none;
        this.CAPTURING_PHASE = EventPhaseEnum.capturing;
        this.AT_TARGET = EventPhaseEnum.atTarget;
        this.BUBBLING_PHASE = EventPhaseEnum.bubbling;
        this[_a] = false;
        this[_b] = false;
        this[_c] = null;
        this[_d] = null;
        this[_e] = false;
        this.type = type;
        this.bubbles = eventInit?.bubbles ?? false;
        this.cancelable = eventInit?.cancelable ?? false;
        this.composed = eventInit?.composed ?? false;
    }
    /**
     * Returns target.
     *
     * @returns Target.
     */
    get target() {
        return this[PropertySymbol.target];
    }
    /**
     * Returns target.
     *
     * @returns Target.
     */
    get currentTarget() {
        return this[PropertySymbol.currentTarget];
    }
    /**
     * Returns "true" if propagation has been stopped.
     *
     * @returns "true" if propagation has been stopped.
     */
    get cancelBubble() {
        return this[PropertySymbol.propagationStopped];
    }
    /**
     * Returns composed path.
     *
     * @returns Composed path.
     */
    composedPath() {
        if (!this[PropertySymbol.target]) {
            return [];
        }
        const composedPath = [];
        let eventTarget = this[PropertySymbol.target];
        while (eventTarget) {
            composedPath.push(eventTarget);
            if (eventTarget.parentNode) {
                eventTarget = eventTarget.parentNode;
            }
            else if (this.composed &&
                eventTarget[PropertySymbol.nodeType] === NodeTypeEnum.documentFragmentNode &&
                eventTarget.host) {
                eventTarget = eventTarget.host;
            }
            else if (eventTarget[PropertySymbol.nodeType] === NodeTypeEnum.documentNode) {
                eventTarget = eventTarget[PropertySymbol.ownerWindow];
            }
            else {
                break;
            }
        }
        return composedPath;
    }
    /**
     * Init event.
     *
     * @deprecated
     * @param type Type.
     * @param [bubbles=false] "true" if it bubbles.
     * @param [cancelable=false] "true" if it cancelable.
     */
    initEvent(type, bubbles = false, cancelable = false) {
        this.type = type;
        this.bubbles = bubbles;
        this.cancelable = cancelable;
    }
    /**
     * Prevents default.
     */
    preventDefault() {
        if (!this[PropertySymbol.isInPassiveEventListener]) {
            this.defaultPrevented = true;
        }
    }
    /**
     * Stops immediate propagation.
     */
    stopImmediatePropagation() {
        this[PropertySymbol.immediatePropagationStopped] = true;
    }
    /**
     * Stops propagation.
     */
    stopPropagation() {
        this[PropertySymbol.propagationStopped] = true;
    }
}
_a = PropertySymbol.immediatePropagationStopped, _b = PropertySymbol.propagationStopped, _c = PropertySymbol.target, _d = PropertySymbol.currentTarget, _e = PropertySymbol.isInPassiveEventListener;
export default Event;
//# sourceMappingURL=Event.js.map