export namespace compress {
	
	export class Suggestion {
	    sizeMB?: number;
	    minutesEach?: number;
	    parts?: number;
	
	    static createFrom(source: any = {}) {
	        return new Suggestion(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sizeMB = source["sizeMB"];
	        this.minutesEach = source["minutesEach"];
	        this.parts = source["parts"];
	    }
	}
	export class Advice {
	    ok: boolean;
	    kbps: number;
	    minKbps: number;
	    message: string;
	    suggestion: Suggestion;
	
	    static createFrom(source: any = {}) {
	        return new Advice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.kbps = source["kbps"];
	        this.minKbps = source["minKbps"];
	        this.message = source["message"];
	        this.suggestion = this.convertValues(source["suggestion"], Suggestion);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Job {
	    inputPath: string;
	    outputPath: string;
	    presetId: string;
	    sizeMB: number;
	    crf: number;
	    split?: split.Spec;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.inputPath = source["inputPath"];
	        this.outputPath = source["outputPath"];
	        this.presetId = source["presetId"];
	        this.sizeMB = source["sizeMB"];
	        this.crf = source["crf"];
	        this.split = this.convertValues(source["split"], split.Spec);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class TaskStatus {
	    jobId: string;
	    kind: string;
	    label: string;
	    inputPath: string;
	    presetId: string;
	    state: string;
	    position: number;
	    percent: number;
	    stage: string;
	    outputPath: string;
	    sizeBytes: number;
	    partsTotal: number;
	    partsDone: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new TaskStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.inputPath = source["inputPath"];
	        this.presetId = source["presetId"];
	        this.state = source["state"];
	        this.position = source["position"];
	        this.percent = source["percent"];
	        this.stage = source["stage"];
	        this.outputPath = source["outputPath"];
	        this.sizeBytes = source["sizeBytes"];
	        this.partsTotal = source["partsTotal"];
	        this.partsDone = source["partsDone"];
	        this.error = source["error"];
	    }
	}

}

export namespace config {
	
	export class Config {
	    presetId: string;
	    sizeMB: number;
	    crf: number;
	    outputDir: string;
	    ffmpegPath: string;
	    ffprobePath: string;
	    language: string;
	    maxParallel: number;
	    notifyOnDone: boolean;
	    openFolderOnDone: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.presetId = source["presetId"];
	        this.sizeMB = source["sizeMB"];
	        this.crf = source["crf"];
	        this.outputDir = source["outputDir"];
	        this.ffmpegPath = source["ffmpegPath"];
	        this.ffprobePath = source["ffprobePath"];
	        this.language = source["language"];
	        this.maxParallel = source["maxParallel"];
	        this.notifyOnDone = source["notifyOnDone"];
	        this.openFolderOnDone = source["openFolderOnDone"];
	    }
	}

}

export namespace estimate {
	
	export class Result {
	    mode: string;
	    available: boolean;
	    targetSizeMB: number;
	    currentSizeMB: number;
	    savedPct: number;
	    videoKbps: number;
	    audioKbps: number;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.available = source["available"];
	        this.targetSizeMB = source["targetSizeMB"];
	        this.currentSizeMB = source["currentSizeMB"];
	        this.savedPct = source["savedPct"];
	        this.videoKbps = source["videoKbps"];
	        this.audioKbps = source["audioKbps"];
	    }
	}

}

export namespace main {
	
	export class JobAck {
	    jobId: string;
	    position: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new JobAck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.position = source["position"];
	        this.error = source["error"];
	    }
	}
	export class SystemStatus {
	    ffmpegOK: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ffmpegOK = source["ffmpegOK"];
	        this.message = source["message"];
	    }
	}

}

export namespace media {
	
	export class Info {
	    path: string;
	    durationSec: number;
	    width: number;
	    height: number;
	    hasAudio: boolean;
	    hasVideo: boolean;
	    sizeMB: number;
	    videoCodec: string;
	    audioCodec: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.durationSec = source["durationSec"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.hasAudio = source["hasAudio"];
	        this.hasVideo = source["hasVideo"];
	        this.sizeMB = source["sizeMB"];
	        this.videoCodec = source["videoCodec"];
	        this.audioCodec = source["audioCodec"];
	    }
	}

}

export namespace presets {
	
	export class Preset {
	    id: string;
	    name: string;
	    description: string;
	    mode: string;
	    sizeMB: number;
	    crf: number;
	    audioBitrate: string;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.mode = source["mode"];
	        this.sizeMB = source["sizeMB"];
	        this.crf = source["crf"];
	        this.audioBitrate = source["audioBitrate"];
	    }
	}

}

export namespace split {
	
	export class Spec {
	    parts: number;
	    minutesEach: number;
	
	    static createFrom(source: any = {}) {
	        return new Spec(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.parts = source["parts"];
	        this.minutesEach = source["minutesEach"];
	    }
	}

}

export namespace sysinfo {
	
	export class Snapshot {
	    cpu: number;
	    memUsedMB: number;
	    memTotalMB: number;
	    ffmpegCpu: number;
	    gpu: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpu = source["cpu"];
	        this.memUsedMB = source["memUsedMB"];
	        this.memTotalMB = source["memTotalMB"];
	        this.ffmpegCpu = source["ffmpegCpu"];
	        this.gpu = source["gpu"];
	    }
	}

}

