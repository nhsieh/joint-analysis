export interface Category {
  id: string;
  name: string;
  description?: string;
  color?: string;
}

export interface Person {
  id: string;
  name: string;
  email?: string;
  created_at: string;
  updated_at: string;
}

export interface Transaction {
  id: string;
  description: string;
  amount: number;
  assigned_to: string[];
  date_uploaded: string;
  file_name: string;
  transaction_date: string;
  posted_date: string;
  card_number: string;
  category_id: string;
}

export interface Archive {
  id: string;
  description?: string;
  archived_at: string;
  transaction_count: number;
  total_amount: number;
  person_totals?: PersonTotal[];
  created_at: string;
  updated_at: string;
}

export interface PersonTotal {
  name: string;
  total: number;
}
