import { Component, OnInit, inject, signal } from '@angular/core';
import { FormBuilder, FormsModule, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { marked } from 'marked';
import { API_URL } from '../../core/api-url';

interface TestCase { id: number; problem_id: number; input: string; expected: string; }

@Component({
  selector: 'app-problem-edit',
  imports: [ReactiveFormsModule, FormsModule],
  template: `
    <div class="page-header">
      <h2>{{ isNew ? 'New Problem' : 'Edit Problem' }}</h2>
      <button class="btn-sm" (click)="router.navigate(['/problems'])">← Back</button>
    </div>

    <form [formGroup]="form" (ngSubmit)="save()">
      <div class="form-grid">
        <label>
          Title
          <input type="text" formControlName="title" />
        </label>
        <label>
          Difficulty
          <select formControlName="difficulty">
            <option>Easy</option>
            <option>Medium</option>
            <option>Hard</option>
          </select>
        </label>
        <label>
          Tags (comma-separated)
          <input type="text" formControlName="tags" />
        </label>
        <label>
          Time Limit (ms)
          <input type="number" formControlName="time_limit_ms" />
        </label>
        <label>
          Memory Limit (MB)
          <input type="number" formControlName="memory_limit_mb" />
        </label>
      </div>

      <div class="description-section">
        <div class="description-editor">
          <label>Description (Markdown)</label>
          <textarea formControlName="description" rows="14" (input)="updatePreview()"></textarea>
        </div>
        <div class="description-preview">
          <label>Preview</label>
          <div class="preview-body" [innerHTML]="preview()"></div>
        </div>
      </div>

      @if (saveError()) { <p class="error">{{ saveError() }}</p> }

      <div class="form-actions">
        <button type="submit" class="btn-primary" [disabled]="saving()">
          {{ saving() ? 'Saving…' : 'Save Problem' }}
        </button>
      </div>
    </form>

    @if (!isNew) {
      <section class="testcases-section">
        <h3>Test Cases</h3>

        @for (tc of testCases(); track tc.id) {
          <div class="tc-card">
            <div class="tc-fields">
              <label>
                Input
                <textarea [(ngModel)]="tc.input" [ngModelOptions]="{standalone: true}" rows="3"></textarea>
              </label>
              <label>
                Expected Output
                <textarea [(ngModel)]="tc.expected" [ngModelOptions]="{standalone: true}" rows="3"></textarea>
              </label>
            </div>
            <div class="tc-actions">
              <button class="btn-sm" (click)="updateTestCase(tc)">Save</button>
              <button class="btn-sm btn-danger" (click)="deleteTestCase(tc.id)">Delete</button>
            </div>
          </div>
        }

        <div class="tc-card tc-new">
          <div class="tc-fields">
            <label>
              Input
              <textarea [(ngModel)]="newTCInput" rows="3" placeholder="(leave blank for no input)"></textarea>
            </label>
            <label>
              Expected Output
              <textarea [(ngModel)]="newTCExpected" rows="3" placeholder="exact expected stdout"></textarea>
            </label>
          </div>
          <button class="btn-sm btn-primary" (click)="addTestCase()">Add Test Case</button>
        </div>

        @if (tcError()) { <p class="error">{{ tcError() }}</p> }
      </section>
    }
  `,
  styleUrl: './problem-edit.component.css',
})
export class ProblemEditComponent implements OnInit {
  private fb = inject(FormBuilder);
  private http = inject(HttpClient);
  private route = inject(ActivatedRoute);
  readonly router = inject(Router);

  problemID: number | null = null;
  get isNew() { return this.problemID === null; }

  form = this.fb.nonNullable.group({
    title: ['', Validators.required],
    description: ['', Validators.required],
    difficulty: ['Easy', Validators.required],
    tags: [''],
    time_limit_ms: [2000, [Validators.required, Validators.min(100)]],
    memory_limit_mb: [256, [Validators.required, Validators.min(16)]],
  });

  preview = signal('');
  saving = signal(false);
  saveError = signal('');

  testCases = signal<TestCase[]>([]);
  newTCInput = '';
  newTCExpected = '';
  tcError = signal('');

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id && id !== 'new') {
      this.problemID = +id;
      this.loadProblem();
      this.loadTestCases();
    }
  }

  private async loadProblem(): Promise<void> {
    try {
      const p: any = await firstValueFrom(this.http.get(`${API_URL}/admin/problems/${this.problemID}`));
      this.form.patchValue({
        title: p.title,
        description: p.description,
        difficulty: p.difficulty,
        tags: (p.tags ?? []).join(', '),
        time_limit_ms: p.time_limit_ms,
        memory_limit_mb: p.memory_limit_mb,
      });
      this.updatePreview();
    } catch {
      this.saveError.set('Failed to load problem.');
    }
  }

  private async loadTestCases(): Promise<void> {
    try {
      const data = await firstValueFrom(this.http.get<TestCase[]>(`${API_URL}/admin/problems/${this.problemID}/testcases`));
      this.testCases.set(data ?? []);
    } catch {
      this.tcError.set('Failed to load test cases.');
    }
  }

  updatePreview(): void {
    const md = this.form.getRawValue().description;
    this.preview.set(marked(md) as string);
  }

  async save(): Promise<void> {
    if (this.form.invalid) return;
    this.saving.set(true);
    this.saveError.set('');
    const v = this.form.getRawValue();
    const body = {
      title: v.title,
      description: v.description,
      difficulty: v.difficulty,
      tags: v.tags ? v.tags.split(',').map(t => t.trim()).filter(Boolean) : [],
      time_limit_ms: v.time_limit_ms,
      memory_limit_mb: v.memory_limit_mb,
    };
    try {
      if (this.isNew) {
        const p: any = await firstValueFrom(this.http.post(`${API_URL}/admin/problems`, body));
        this.router.navigate(['/problems', p.id, 'edit']);
      } else {
        await firstValueFrom(this.http.put(`${API_URL}/admin/problems/${this.problemID}`, body));
      }
    } catch {
      this.saveError.set('Failed to save problem.');
    } finally {
      this.saving.set(false);
    }
  }

  async addTestCase(): Promise<void> {
    if (!this.newTCExpected) { this.tcError.set('Expected output is required.'); return; }
    this.tcError.set('');
    try {
      const tc = await firstValueFrom(
        this.http.post<TestCase>(`${API_URL}/admin/problems/${this.problemID}/testcases`, {
          input: this.newTCInput,
          expected: this.newTCExpected,
        })
      );
      this.testCases.update(list => [...list, tc]);
      this.newTCInput = '';
      this.newTCExpected = '';
    } catch {
      this.tcError.set('Failed to add test case.');
    }
  }

  async updateTestCase(tc: TestCase): Promise<void> {
    this.tcError.set('');
    try {
      await firstValueFrom(
        this.http.put(`${API_URL}/admin/testcases/${tc.id}`, { input: tc.input, expected: tc.expected })
      );
    } catch {
      this.tcError.set('Failed to update test case.');
    }
  }

  async deleteTestCase(id: number): Promise<void> {
    if (!confirm('Delete this test case?')) return;
    try {
      await firstValueFrom(this.http.delete(`${API_URL}/admin/testcases/${id}`));
      this.testCases.update(list => list.filter(tc => tc.id !== id));
    } catch {
      this.tcError.set('Failed to delete test case.');
    }
  }
}
