/* tslint:disable:no-unused-variable */

import {TestBed, getTestBed, tick, fakeAsync} from '@angular/core/testing';
import {VariableService} from '../../../service/variable/variable.service';
import {TranslateService, TranslateLoader, TranslateParser, TranslateModule} from '@ngx-translate/core';
import {SharedService} from '../../shared.service';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector} from '@angular/core';
import {VariableFormComponent} from './variable.form';
import {GroupService} from '../../../service/group/group.service';
import {Variable} from '../../../model/variable.model';
import {VariableEvent} from '../variable.event.model';
import {SharedModule} from '../../shared.module';
import {HttpClientTestingModule} from '@angular/common/http/testing';
import { ParameterService } from 'app/service/parameter/parameter.service';
import { Observable, of } from 'rxjs';

describe('CDS: Variable From Component', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [
            ],
            providers: [
                { provide: VariableService, useClass: MockApplicationService },
                GroupService,
                SharedService,
                TranslateService,
                TranslateLoader,
                TranslateParser
            ],
            imports : [
                SharedModule,
                TranslateModule.forRoot(),
                RouterTestingModule.withRoutes([]),
                HttpClientTestingModule
            ]
        });
        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });


    it('Create new variable', fakeAsync( () => {

        // Create component
        let fixture = TestBed.createComponent(VariableFormComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.detectChanges();
        tick(50);

        expect(fixture.debugElement.nativeElement.querySelector('.ui.button.disabled')).toBeTruthy();

        let compiled = fixture.debugElement.nativeElement;

        let variable = new Variable();
        variable.name = 'foo';
        variable.type = 'string';
        variable.value = 'bar';

        fixture.detectChanges();
        tick(50);

        // simulate typing new variable
        let inputName = compiled.querySelector('input[name="name"]');
        inputName.value = variable.name;
        inputName.dispatchEvent(new Event('input'));

        fixture.componentInstance.newVariable.type = variable.type;

        fixture.detectChanges();
        tick(50);

        let inputValue = compiled.querySelector('input[name="value"]');
        inputValue.value = variable.value;
        inputValue.dispatchEvent(new Event('input'));
        inputValue.dispatchEvent(new Event('change'));

        spyOn(fixture.componentInstance.createVariableEvent, 'emit');
        compiled.querySelector('.ui.green.button').click();

        expect(fixture.componentInstance.createVariableEvent.emit).toHaveBeenCalledWith(new VariableEvent('add', variable));
    }));
});

class MockApplicationService {
    constructor() { }

    getTypesFromCache(): string[] {
        return [];
    }

    getTypesFromAPI(): Observable<string[]> {
        return of(["string", "password"])
    }
}