import { Component, inject, signal, OnInit } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { API_URL } from '../../core/api-url';

interface Problem {
  id: number;
  title: string;
  difficulty: string;
  tags: string[];
  time_limit_ms: number;
  memory_limit_mb: number;
}

@Component({
  selector: 'app-problems',
  imports: [RouterLink],
  template: `
    <div class="page-header">
      <h2>Problems</h2>
      <a routerLink="/problems/new" class="btn-primary">+ New Problem</a>
    </div>

    @if (loading()) {
      <p class="muted">Loading…</p>
    } @else if (problems().length === 0) {
      <p class="muted">No problems yet.</p>
    } @else {
      <table class="data-table">
        <thead>
          <tr>
            <th>#</th>
            <th>Title</th>
            <th>Difficulty</th>
            <th>Tags</th>
            <th>Time (ms)</th>
            <th>Memory (MB)</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          @for (p of problems(); track p.id) {
            <tr>
              <td>{{ p.id }}</td>
              <td>{{ p.title }}</td>
              <td><span class="badge badge-{{ p.difficulty.toLowerCase() }}">{{ p.difficulty }}</span></td>
              <td>{{ p.tags.join(', ') }}</td>
              <td>{{ p.time_limit_ms }}</td>
              <td>{{ p.memory_limit_mb }}</td>
              <td class="actions">
                <a [routerLink]="['/problems', p.id, 'edit']" class="btn-sm">Edit</a>
                <button class="btn-sm btn-danger" (click)="deleteProblem(p.id)">Delete</button>
              </td>
            </tr>
          }
        </tbody>
      </table>
    }

    @if (error()) {
      <p class="error">{{ error() }}</p>
    }
  `,
  styleUrl: './problems.component.css',
})
export class ProblemsComponent implements OnInit {
  private http = inject(HttpClient);
  private router = inject(Router);

  problems = signal<Problem[]>([]);
  loading = signal(true);
  error = signal('');

  ngOnInit(): void {
    this.load();
  }

  private async load(): Promise<void> {
    try {
      const data = await firstValueFrom(this.http.get<Problem[]>(`${API_URL}/admin/problems`));
      this.problems.set(data ?? []);
    } catch {
      this.error.set('Failed to load problems.');
    } finally {
      this.loading.set(false);
    }
  }

  async deleteProblem(id: number): Promise<void> {
    if (!confirm('Delete this problem and all its test cases?')) return;
    try {
      await firstValueFrom(this.http.delete(`${API_URL}/admin/problems/${id}`));
      this.problems.update(list => list.filter(p => p.id !== id));
    } catch {
      this.error.set('Failed to delete problem.');
    }
  }
}
