import MouseEvent from './MouseEvent.js';
/**
 *
 */
export default class PointerEvent extends MouseEvent {
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.getCoalescedEvents = () => this.coalescedEvents;
        this.getPredictedEvents = () => this.predictedEvents;
        this.pointerId = eventInit?.pointerId ?? 0;
        this.width = eventInit?.width ?? 1;
        this.height = eventInit?.height ?? 1;
        this.pressure = eventInit?.pressure ?? 0;
        this.tangentialPressure = eventInit?.tangentialPressure ?? 0;
        this.tiltX = eventInit?.tiltX ?? 0;
        this.tiltY = eventInit?.tiltY ?? 0;
        this.twist = eventInit?.twist ?? 0;
        this.altitudeAngle = eventInit?.altitudeAngle ?? 0;
        this.azimuthAngle = eventInit?.azimuthAngle ?? 0;
        this.pointerType = eventInit?.pointerType ?? '';
        this.isPrimary = eventInit?.isPrimary ?? false;
        this.coalescedEvents = eventInit?.coalescedEvents ?? [];
        this.predictedEvents = eventInit?.predictedEvents ?? [];
    }
}
//# sourceMappingURL=PointerEvent.js.map