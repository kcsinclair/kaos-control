"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Document ready state manager.
 */
class DocumentReadyStateManager {
    /**
     * Constructor.
     *
     * @param window
     */
    constructor(window) {
        this.totalTasks = 0;
        this.readyStateCallbacks = [];
        this.window = null;
        this.immediate = null;
        this.isComplete = false;
        this.window = window;
    }
    /**
     * Returns a promise that is fulfilled when ready state is complete.
     *
     * @returns Promise.
     */
    waitUntilComplete() {
        return new Promise((resolve) => {
            if (this.isComplete) {
                resolve();
            }
            else {
                this.readyStateCallbacks.push(resolve);
                if (this.totalTasks === 0 && !this.immediate) {
                    this.immediate = this.window.requestAnimationFrame(this.endTask.bind(this));
                }
            }
        });
    }
    /**
     * Starts a task.
     */
    startTask() {
        if (this.isComplete) {
            return;
        }
        if (this.immediate) {
            this.window.cancelAnimationFrame(this.immediate);
            this.immediate = null;
        }
        this.totalTasks++;
    }
    /**
     * Ends a task.
     */
    endTask() {
        if (this.isComplete) {
            return;
        }
        if (this.immediate) {
            this.window.cancelAnimationFrame(this.immediate);
            this.immediate = null;
        }
        this.totalTasks--;
        if (this.totalTasks <= 0) {
            const callbacks = this.readyStateCallbacks;
            this.readyStateCallbacks = [];
            this.isComplete = true;
            for (const callback of callbacks) {
                callback();
            }
        }
    }
}
exports.default = DocumentReadyStateManager;
//# sourceMappingURL=DocumentReadyStateManager.cjs.map