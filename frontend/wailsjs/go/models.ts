export namespace api {
	
	export class Task {
	    id: string;
	    title: string;
	    filename: string;
	    size: number;
	    path: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.filename = source["filename"];
	        this.size = source["size"];
	        this.path = source["path"];
	        this.status = source["status"];
	    }
	}
	export class TaskListResult {
	    tasks: Task[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new TaskListResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tasks = this.convertValues(source["tasks"], Task);
	        this.total = source["total"];
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

}

export namespace main {
	
	export class ServiceStatus {
	    running: boolean;
	    proxyOn: boolean;
	    apiUrl: string;
	    proxyPort: number;
	    localIps: string[];
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.proxyOn = source["proxyOn"];
	        this.apiUrl = source["apiUrl"];
	        this.proxyPort = source["proxyPort"];
	        this.localIps = source["localIps"];
	    }
	}

}

