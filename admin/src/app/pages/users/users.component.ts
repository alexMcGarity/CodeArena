import { Component, OnInit, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { API_URL } from '../../core/api-url';

interface UserSummary { id: number; email: string; role: string; }

@Component({
  selector: 'app-users',
  template: `
    <h2>Users</h2>

    @if (loading()) {
      <p class="muted">Loading…</p>
    } @else {
      <table class="data-table">
        <thead>
          <tr><th>#</th><th>Email</th><th>Role</th></tr>
        </thead>
        <tbody>
          @for (u of users(); track u.id) {
            <tr>
              <td>{{ u.id }}</td>
              <td>{{ u.email }}</td>
              <td><span class="badge badge-{{ u.role }}">{{ u.role }}</span></td>
            </tr>
          }
        </tbody>
      </table>
    }

    @if (error()) { <p class="error">{{ error() }}</p> }
  `,
  styleUrl: './users.component.css',
})
export class UsersComponent implements OnInit {
  private http = inject(HttpClient);
  users = signal<UserSummary[]>([]);
  loading = signal(true);
  error = signal('');

  ngOnInit(): void { this.load(); }

  private async load(): Promise<void> {
    try {
      const data = await firstValueFrom(this.http.get<UserSummary[]>(`${API_URL}/admin/users`));
      this.users.set(data ?? []);
    } catch {
      this.error.set('Failed to load users.');
    } finally {
      this.loading.set(false);
    }
  }
}
