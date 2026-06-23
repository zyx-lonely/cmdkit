export namespace main {
	
	export class Command {
	    name: string;
	    description: string;
	    syntax: string;
	    examples: string[];
	    difficulty: string;
	    related: string[];
	    scenario: string;
	    platforms: string[];
	    altFor: string;
	
	    static createFrom(source: any = {}) {
	        return new Command(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.syntax = source["syntax"];
	        this.examples = source["examples"];
	        this.difficulty = source["difficulty"];
	        this.related = source["related"];
	        this.scenario = source["scenario"];
	        this.platforms = source["platforms"];
	        this.altFor = source["altFor"];
	    }
	}
	export class Category {
	    name: string;
	    icon: string;
	    role: string;
	    commands: Command[];
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.role = source["role"];
	        this.commands = this.convertValues(source["commands"], Command);
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
	
	export class DistroInfo {
	    id: string;
	    name: string;
	    pretty: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new DistroInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.pretty = source["pretty"];
	        this.version = source["version"];
	    }
	}
	export class ExecResult {
	    success: boolean;
	    output: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ExecResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.output = source["output"];
	        this.error = source["error"];
	    }
	}
	export class FetchResult {
	    success: boolean;
	    content: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new FetchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.content = source["content"];
	        this.error = source["error"];
	    }
	}
	export class GuideStep {
	    step: string;
	
	    static createFrom(source: any = {}) {
	        return new GuideStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step = source["step"];
	    }
	}
	export class InstallGuide {
	    name: string;
	    description: string;
	    url: string;
	    steps: GuideStep[];
	    tips: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new InstallGuide(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.url = source["url"];
	        this.steps = this.convertValues(source["steps"], GuideStep);
	        this.tips = source["tips"];
	        this.note = source["note"];
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
	export class GuideCategory {
	    name: string;
	    icon: string;
	    guides: InstallGuide[];
	
	    static createFrom(source: any = {}) {
	        return new GuideCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.guides = this.convertValues(source["guides"], InstallGuide);
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
	
	
	export class SysInfo {
	    os: string;
	    hostname: string;
	    kernel: string;
	    cpu: string;
	    cores: string;
	    memory: string;
	    disk: string;
	    uptime: string;
	    goVersion: string;
	    shell: string;
	    desktop: string;
	
	    static createFrom(source: any = {}) {
	        return new SysInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.hostname = source["hostname"];
	        this.kernel = source["kernel"];
	        this.cpu = source["cpu"];
	        this.cores = source["cores"];
	        this.memory = source["memory"];
	        this.disk = source["disk"];
	        this.uptime = source["uptime"];
	        this.goVersion = source["goVersion"];
	        this.shell = source["shell"];
	        this.desktop = source["desktop"];
	    }
	}
	export class SysStats {
	    cpuUsage: string;
	    memTotal: string;
	    memUsed: string;
	    memPct: string;
	    diskTotal: string;
	    diskUsed: string;
	    diskPct: string;
	    time: string;
	
	    static createFrom(source: any = {}) {
	        return new SysStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cpuUsage = source["cpuUsage"];
	        this.memTotal = source["memTotal"];
	        this.memUsed = source["memUsed"];
	        this.memPct = source["memPct"];
	        this.diskTotal = source["diskTotal"];
	        this.diskUsed = source["diskUsed"];
	        this.diskPct = source["diskPct"];
	        this.time = source["time"];
	    }
	}

}

