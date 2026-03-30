// RunPipeline.tsx — Pipeline visualization for dispatched intents

import {
  ArrowRight,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Loader2,
  Send,
  X,
  XCircle,
} from 'lucide-react'
import { useState } from 'react'
import type { PipelineRun, PipelineStage } from '../types'

interface RunPipelineProps {
  runs: PipelineRun[]
  activeRun: PipelineRun | null
  onDismiss: (runId: string) => void
}

const STAGE_LABEL: Record<PipelineStage, string> = {
  outflow: 'Outflow',
  processing: 'Processing',
  return: 'Return',
}

const STAGES: PipelineStage[] = ['outflow', 'processing', 'return']

function StageIcon({ stage, run }: { stage: PipelineStage; run: PipelineRun }) {
  const stageIndex = STAGES.indexOf(stage)
  const runStageIndex = STAGES.indexOf(run.stage)
  const reached = stageIndex <= runStageIndex
  const isCurrent = stage === run.stage

  if (!reached) {
    return <span className="is-pipeline__stage-icon dimmed" />
  }
  if (stage === 'return' && isCurrent) {
    return run.status === 'success' ? (
      <CheckCircle2 size={14} className="is-pipeline__stage-icon success" />
    ) : (
      <XCircle size={14} className="is-pipeline__stage-icon failed" />
    )
  }
  if (isCurrent && run.status === 'running') {
    return <Loader2 size={14} className="is-pipeline__stage-icon running" />
  }
  if (stage === 'outflow') {
    return <Send size={14} className="is-pipeline__stage-icon done" />
  }
  return <CheckCircle2 size={14} className="is-pipeline__stage-icon done" />
}

function formatTime(ts: string | null): string {
  if (!ts) return ''
  return new Date(ts).toLocaleTimeString()
}

export function RunPipeline({ runs, activeRun, onDismiss }: RunPipelineProps) {
  const [expanded, setExpanded] = useState(false)
  const run = activeRun ?? runs[0]
  if (!run) return null

  const isActive = run.status === 'running' || run.status === 'pending'
  const runStageIndex = STAGES.indexOf(run.stage)

  return (
    <div className="is-pipeline" data-status={run.status}>
      <div className="is-pipeline__header">
        <span className="is-pipeline__intent">{run.intent}</span>
        <button
          type="button"
          className="is-pipeline__dismiss"
          onClick={() => onDismiss(run.run_id)}
          title="Dismiss"
        >
          <X size={12} />
        </button>
      </div>

      <div className="is-pipeline__stages">
        {STAGES.map((stage, i) => {
          const stageIndex = STAGES.indexOf(stage)
          const passed = stageIndex < runStageIndex
          const isFlowing = isActive && stageIndex <= runStageIndex

          return (
            <div key={stage} className="is-pipeline__stage-group">
              <div
                className={`is-pipeline__stage ${stage === run.stage ? 'current' : ''} ${passed ? 'passed' : ''}`}
                data-status={stage === run.stage ? run.status : passed ? 'done' : 'pending'}
              >
                <StageIcon stage={stage} run={run} />
                <span className="is-pipeline__stage-label">{STAGE_LABEL[stage]}</span>
              </div>
              {i < STAGES.length - 1 && (
                <div className={`is-pipeline__connector ${isFlowing ? 'flow' : ''}`}>
                  <ArrowRight size={10} />
                </div>
              )}
            </div>
          )
        })}
      </div>

      <div className="is-pipeline__meta">
        <span className="is-pipeline__badge" data-strategy={run.routing.strategy}>
          {run.routing.strategy}
        </span>
        <span className="is-pipeline__executor">{run.routing.executor}</span>
        {run.routing.cap_name && <span className="is-pipeline__cap">{run.routing.cap_name}</span>}
        {run.events.length > 0 && (
          <span className="is-pipeline__event-count">{run.events.length} events</span>
        )}
      </div>

      {run.result && <div className={`is-pipeline__result ${run.status}`}>{run.result}</div>}

      {run.events.length > 0 && (
        <div className="is-pipeline__detail-toggle">
          <button
            type="button"
            className="is-pipeline__expand-btn"
            onClick={() => setExpanded(!expanded)}
          >
            {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
            <span>Timeline</span>
          </button>
        </div>
      )}

      {expanded && run.events.length > 0 && (
        <div className="is-pipeline__detail">
          {run.events.map((ev) => (
            <div key={ev.id} className="is-pipeline__event">
              <span className="is-pipeline__event-time">{formatTime(ev.created_at)}</span>
              <span className="is-pipeline__event-dot" data-type={ev.event_type} />
              <span className="is-pipeline__event-text">{ev.summary}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
