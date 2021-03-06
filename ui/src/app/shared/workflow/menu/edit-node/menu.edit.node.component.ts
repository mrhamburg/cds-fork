import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output
} from '@angular/core';
import { Store } from '@ngxs/store';
import { IPopup } from '@richardlt/ng2-semantic-ui';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';

@Component({
    selector: 'app-workflow-menu-wnode-edit',
    templateUrl: './menu.edit.node.html',
    styleUrls: ['./menu.edit.node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWNodeMenuEditComponent implements OnInit {

    // Project that contains the workflow
    @Input() popup: IPopup;
    @Output() event = new EventEmitter<string>();

    project: Project;
    workflow: Workflow;
    workflowrun: WorkflowRun;
    noderun: WorkflowNodeRun;
    node: WNode;
    runnable: boolean;
    readonly = true;

    constructor(
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {}

    ngOnInit(): void {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);

        let state: WorkflowStateModel = this._store.selectSnapshot(WorkflowState);
        this.workflow = state.workflow;
        this.workflowrun = state.workflowRun;
        this.noderun = state.workflowNodeRun;
        this.node = state.node;
        this.readonly = !state.canEdit;

        this.runnable = this.getCanBeRun();
        this._cd.markForCheck();
    }

    sendEvent(e: string): void {
        this.popup.close();
        this.event.emit(e);
    }

    getCanBeRun(): boolean {
        if (!this.workflow) {
            return;
        }

        if (this.workflow && !this.workflow.permissions.executable) {
            return false;
        }

        // If we are in a run, check if current node can be run ( compute by cds api)
        if (this.noderun && this.workflowrun && this.workflowrun.nodes) {
            let nodesRun = this.workflowrun.nodes[this.noderun.workflow_node_id];
            if (nodesRun) {
                let nodeRun = nodesRun.find(n => {
                    return n.id === this.noderun.id;
                });
                if (nodeRun) {
                    return nodeRun.can_be_run;
                }
            }
            return false;
        }

        let workflowrunIsNotActive = this.workflowrun && !PipelineStatus.isActive(this.workflowrun.status);
        if (workflowrunIsNotActive && this.noderun) {
            return true;
        }

        if (this.node && this.workflowrun) {
            if (workflowrunIsNotActive && !this.noderun &&
                this.node.id === this.workflowrun.workflow.workflow_data.node.id) {
                return true;
            }

            if (this.workflowrun && this.workflowrun.workflow && this.workflowrun.workflow.workflow_data.node.id > 0) {
                let nbNodeFound = 0;
                let parentNodes = Workflow.getParentNodeIds(this.workflowrun, this.node.id);
                for (let parentNodeId of parentNodes) {
                    for (let nodeRunId in this.workflowrun.nodes) {
                        if (!this.workflowrun.nodes[nodeRunId]) {
                            continue;
                        }
                        let nodeRuns = this.workflowrun.nodes[nodeRunId];
                        if (nodeRuns[0].workflow_node_id === parentNodeId) { // if node id is still the same
                            if (PipelineStatus.isActive(nodeRuns[0].status)) {
                                return false;
                            }
                            nbNodeFound++;
                        } else if (!Workflow.getNodeByID(nodeRuns[0].workflow_node_id, this.workflowrun.workflow)) {
                            // workflow updated so prefer return true
                            return true;
                        }
                    }
                }
                if (nbNodeFound !== parentNodes.length) { // It means that a parent node isn't already executed
                    return false;
                }
            }
        }
        return true;
    }
}
