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
var _a, _b, _c, _d, _e;
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../nodes/node/NodeTypeEnum.cjs"));
const EventPhaseEnum_js_1 = __importDefault(require("./EventPhaseEnum.cjs"));
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
        this.eventPhase = EventPhaseEnum_js_1.default.none;
        this.timeStamp = performance.now();
        this.NONE = EventPhaseEnum_js_1.default.none;
        this.CAPTURING_PHASE = EventPhaseEnum_js_1.default.capturing;
        this.AT_TARGET = EventPhaseEnum_js_1.default.atTarget;
        this.BUBBLING_PHASE = EventPhaseEnum_js_1.default.bubbling;
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
                eventTarget[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.documentFragmentNode &&
                eventTarget.host) {
                eventTarget = eventTarget.host;
            }
            else if (eventTarget[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.documentNode) {
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
exports.default = Event;
//# sourceMappingURL=Event.cjs.map