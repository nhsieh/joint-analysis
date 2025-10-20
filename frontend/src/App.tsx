import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';

interface Transaction {
  id: number;
  description: string;
  amount: number;
  assigned_to: string;
  date_uploaded: string;
  file_name: string;
}

interface Person {
  id: number;
  name: string;
}

interface PersonTotal {
  name: string;
  total: number;
}

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

function App() {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [people, setPeople] = useState<Person[]>([]);
  const [totals, setTotals] = useState<PersonTotal[]>([]);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [newPersonName, setNewPersonName] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchTransactions();
    fetchPeople();
    fetchTotals();
  }, []);

  const fetchTransactions = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/transactions`);
      setTransactions(response.data || []);
    } catch (error) {
      console.error('Error fetching transactions:', error);
    }
  };

  const fetchPeople = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/people`);
      setPeople(response.data || []);
    } catch (error) {
      console.error('Error fetching people:', error);
    }
  };

  const fetchTotals = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/totals`);
      setTotals(response.data || []);
    } catch (error) {
      console.error('Error fetching totals:', error);
    }
  };

  const handleFileUpload = async () => {
    if (!selectedFile) {
      alert('Please select a file first');
      return;
    }

    setLoading(true);
    const formData = new FormData();
    formData.append('file', selectedFile);

    try {
      await axios.post(`${API_URL}/api/upload-csv`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      setSelectedFile(null);
      fetchTransactions();
      alert('CSV uploaded successfully!');
    } catch (error) {
      console.error('Error uploading file:', error);
      alert('Error uploading file');
    } finally {
      setLoading(false);
    }
  };

  const assignTransaction = async (transactionId: number, personName: string) => {
    try {
      await axios.put(`${API_URL}/api/transactions/${transactionId}/assign`, {
        assigned_to: personName,
      });
      fetchTransactions();
      fetchTotals();
    } catch (error) {
      console.error('Error assigning transaction:', error);
      alert('Error assigning transaction');
    }
  };

  const createPerson = async () => {
    if (!newPersonName.trim()) {
      alert('Please enter a person name');
      return;
    }

    try {
      await axios.post(`${API_URL}/api/people`, { name: newPersonName });
      setNewPersonName('');
      fetchPeople();
    } catch (error) {
      console.error('Error creating person:', error);
      alert('Error creating person');
    }
  };

  return (
    <div className="App">
      <header className="App-header">
        <h1>Joint Analysis - Expense Tracker</h1>
      </header>

      <main className="container">
        {/* File Upload Section */}
        <section className="upload-section">
          <h2>Upload CSV File</h2>
          <div className="upload-controls">
            <input
              type="file"
              accept=".csv"
              onChange={(e) => setSelectedFile(e.target.files?.[0] || null)}
            />
            <button onClick={handleFileUpload} disabled={loading || !selectedFile}>
              {loading ? 'Uploading...' : 'Upload CSV'}
            </button>
          </div>
        </section>

        {/* Add Person Section */}
        <section className="add-person-section">
          <h2>Add Person</h2>
          <div className="add-person-controls">
            <input
              type="text"
              placeholder="Enter person name"
              value={newPersonName}
              onChange={(e) => setNewPersonName(e.target.value)}
            />
            <button onClick={createPerson}>Add Person</button>
          </div>
        </section>

        {/* Totals Section */}
        <section className="totals-section">
          <h2>Totals by Person</h2>
          <div className="totals-grid">
            {totals.map((total) => (
              <div key={total.name} className="total-card">
                <h3>{total.name}</h3>
                <p className="total-amount">${total.total.toFixed(2)}</p>
              </div>
            ))}
          </div>
        </section>

        {/* Transactions Section */}
        <section className="transactions-section">
          <h2>Transactions</h2>
          <div className="transactions-table">
            <table>
              <thead>
                <tr>
                  <th>Description</th>
                  <th>Amount</th>
                  <th>Assigned To</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {transactions.map((transaction) => (
                  <tr key={transaction.id}>
                    <td>{transaction.description}</td>
                    <td>${transaction.amount.toFixed(2)}</td>
                    <td>{transaction.assigned_to || 'Unassigned'}</td>
                    <td>
                      <select
                        value={transaction.assigned_to || ''}
                        onChange={(e) => assignTransaction(transaction.id, e.target.value)}
                      >
                        <option value="">Select Person</option>
                        {people.map((person) => (
                          <option key={person.id} value={person.name}>
                            {person.name}
                          </option>
                        ))}
                      </select>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </main>
    </div>
  );
}

export default App;