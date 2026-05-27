import { Injectable, computed, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { API_URL } from './api-url';

interface AuthResponse {
  token: string;
  user_id: number;
  email: string;
  role: string;
}

interface AdminUser {
  id: number;
  email: string;
  role: string;
}

@Injectable({ providedIn: 'root' })
export class AuthService {
  private tokenSignal = signal<string | null>(localStorage.getItem('adminToken'));
  private userSignal = signal<AdminUser | null>(
    JSON.parse(localStorage.getItem('adminUser') || 'null')
  );

  readonly token = this.tokenSignal.asReadonly();
  readonly user = this.userSignal.asReadonly();
  readonly isLoggedIn = computed(() => !!this.tokenSignal());
  readonly isAdmin = computed(() => this.userSignal()?.role === 'admin');

  constructor(private http: HttpClient) {}

  async login(email: string, password: string): Promise<void> {
    const data = await firstValueFrom(
      this.http.post<AuthResponse>(`${API_URL}/auth/login`, { email, password })
    );
    if (data.role !== 'admin') {
      throw new Error('This account does not have admin access.');
    }
    this.persist(data);
  }

  logout(): void {
    localStorage.removeItem('adminToken');
    localStorage.removeItem('adminUser');
    this.tokenSignal.set(null);
    this.userSignal.set(null);
  }

  private persist(data: AuthResponse): void {
    localStorage.setItem('adminToken', data.token);
    const user: AdminUser = { id: data.user_id, email: data.email, role: data.role };
    localStorage.setItem('adminUser', JSON.stringify(user));
    this.tokenSignal.set(data.token);
    this.userSignal.set(user);
  }
}
