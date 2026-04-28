import IEventInit from './IEventInit.cjs';
import * as PropertySymbol from '../PropertySymbol.cjs';
import EventTarget from './EventTarget.cjs';
import EventPhaseEnum from './EventPhaseEnum.cjs';
/**
 * Event.
 */
export default class Event {
    composed: boolean;
    bubbles: boolean;
    cancelable: boolean;
    defaultPrevented: boolean;
    eventPhase: EventPhaseEnum;
    timeStamp: number;
    type: string;
    NONE: EventPhaseEnum;
    CAPTURING_PHASE: EventPhaseEnum;
    AT_TARGET: EventPhaseEnum;
    BUBBLING_PHASE: EventPhaseEnum;
    [PropertySymbol.immediatePropagationStopped]: boolean;
    [PropertySymbol.propagationStopped]: boolean;
    [PropertySymbol.target]: EventTarget;
    [PropertySymbol.currentTarget]: EventTarget;
    [PropertySymbol.isInPassiveEventListener]: boolean;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IEventInit | null);
    /**
     * Returns target.
     *
     * @returns Target.
     */
    get target(): EventTarget;
    /**
     * Returns target.
     *
     * @returns Target.
     */
    get currentTarget(): EventTarget;
    /**
     * Returns "true" if propagation has been stopped.
     *
     * @returns "true" if propagation has been stopped.
     */
    get cancelBubble(): boolean;
    /**
     * Returns composed path.
     *
     * @returns Composed path.
     */
    composedPath(): EventTarget[];
    /**
     * Init event.
     *
     * @deprecated
     * @param type Type.
     * @param [bubbles=false] "true" if it bubbles.
     * @param [cancelable=false] "true" if it cancelable.
     */
    initEvent(type: string, bubbles?: boolean, cancelable?: boolean): void;
    /**
     * Prevents default.
     */
    preventDefault(): void;
    /**
     * Stops immediate propagation.
     */
    stopImmediatePropagation(): void;
    /**
     * Stops propagation.
     */
    stopPropagation(): void;
}
//# sourceMappingURL=Event.d.ts.map